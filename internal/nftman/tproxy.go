// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"fmt"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

func (t *NFTManager) addMarkChainForTProxy(
	conn *nftables.Conn, tp *config.TProxy,
) (
	ret *nftables.Chain, err error,
) {
	// nft: chain <tp.Name>-MARK { ... }
	chain := &nftables.Chain{
		Table: t.table,
		Name:  tp.Name + "-MARK",
	}

	conn.AddChain(chain)
	t.addProxyProtocolGuards(conn, chain, tp)
	t.addDNSHijackGuard(conn, chain, tp)

	// nft: meta mark set <tp.Mark>

	exprs := []expr.Any{
		&expr.Immediate{ // immediate reg 1 ...
			Register: 1,
			Data: binaryutil.NativeEndian.PutUint32(
				uint32(tp.Mark),
			),
		},
		&expr.Meta{
			Key:            expr.MetaKeyMARK,
			SourceRegister: true,
			Register:       1,
		},
	}

	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	})

	ret = chain

	return
}

func (t *NFTManager) addTproxyChainForTProxy(
	conn *nftables.Conn, tp *config.TProxy,
) (
	ret *nftables.Chain, err error,
) {
	// nft: chain <tp.Name> { ... }
	chain := &nftables.Chain{
		Table: t.table,
		Name:  tp.Name,
	}

	conn.AddChain(chain)

	// nft: meta l4proto tcp tproxy to :<tp.Port>
	// With UDP enabled: meta l4proto { tcp, udp } tproxy to :<tp.Port>
	// NoIPv6 selects the IPv4-only form: tproxy ip to :<tp.Port>.
	tproxy := &expr.TProxy{ // tproxy port reg 1
		Family:  byte(nftables.TableFamilyUnspecified),
		RegPort: 1,
	}

	exprs := []expr.Any{
		&expr.Meta{ // meta load l4proto => reg 1
			Key:      expr.MetaKeyL4PROTO,
			Register: 1,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00000006
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{unix.IPPROTO_TCP},
		},
		&expr.Immediate{ // immediate reg 1 ...
			Register: 1,
			Data:     binaryutil.BigEndian.PutUint16(tp.Port),
		},
		tproxy,
	}

	lookup := &exprs[1]

	if !tp.NoUDP {
		// NOTE:
		// Only add set when we use it, otherwise we will get an EINVAL
		// https://github.com/torvalds/linux/blob/4f82870119a46b0d04d91ef4697ac4977a255a9d/net/netfilter/nf_tables_api.c#L9881

		t.protoSet.ID = 0
		err = conn.AddSet(t.protoSet, t.protoSetElement)
		if err != nil {
			return nil, fmt.Errorf("create TCP/UDP protocol set: %w", err)
		}
		*lookup = &expr.Lookup{ // lookup reg 1 set __set%d
			SourceRegister: 1,
			SetID:          t.protoSet.ID,
			SetName:        t.protoSet.Name,
		}
	}

	if tp.NoIPv6 {
		tproxy.Family = byte(nftables.TableFamilyIPv4)
	}

	exprs = addDebugCounter(exprs)

	rule := &nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	}

	conn.AddRule(rule)

	ret = chain

	return
}

// Excluded protocols must bypass marking as well as TPROXY dispatch.
func (nft *NFTManager) addProxyProtocolGuards(conn *nftables.Conn, chain *nftables.Chain, tp *config.TProxy) {
	if tp.NoIPv6 {
		// nft: meta nfproto ipv6 return
		nft.addReturnRule(conn, chain,
			&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
			&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV6}},
		)
	}
	if tp.NoUDP {
		if tp.DNSHijack != nil && !tp.NoIPv6 {
			// DNSHijack only supports IPv4; IPv6 UDP must still bypass.
			// nft: meta nfproto ipv6 meta l4proto udp return
			nft.addReturnRule(conn, chain,
				&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
				&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV6}},
				&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
				&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.IPPROTO_UDP}},
			)
		}
		// nft without DNS hijacking: meta l4proto udp return
		// With DNS hijacking, append the port condition below so IPv4
		// UDP/53 can reach DNS DNAT despite NoUDP.
		filters := []expr.Any{
			&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
			&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.IPPROTO_UDP}},
		}
		if tp.DNSHijack != nil {
			// nft: meta l4proto udp udp dport != 53 return
			filters = append(filters,
				&expr.Payload{DestRegister: 1, Base: expr.PayloadBaseTransportHeader, Offset: 2, Len: 2},
				&expr.Cmp{Op: expr.CmpOpNeq, Register: 1, Data: binaryutil.BigEndian.PutUint16(53)},
			)
		}
		nft.addReturnRule(conn, chain, filters...)
	}
}

func (nft *NFTManager) initMarkMap(conn *nftables.Conn) (err error) {
	// nft: map mark-vmap { type mark : verdict; }
	nft.markTproxyMap = &nftables.Set{
		Table:        nft.table,
		Name:         "mark-vmap",
		KeyType:      nftables.TypeMark,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	err = conn.AddSet(nft.markTproxyMap, []nftables.SetElement{})
	if err != nil {
		return fmt.Errorf("create map %q: %w", nft.markTproxyMap.Name, err)
	}

	return
}

func (t *NFTManager) updateMarkTproxyMap(
	conn *nftables.Conn, mark config.FireWallMark, chain string,
) (
	err error,
) {
	// nft: add element inet cgtproxy mark-vmap { <mark> : goto <chain> }
	setElement := nftables.SetElement{
		Key: binaryutil.NativeEndian.PutUint32(uint32(mark)),
		VerdictData: &expr.Verdict{
			Kind:  expr.VerdictGoto,
			Chain: chain,
		},
	}
	err = conn.SetAddElements(
		t.markTproxyMap,
		[]nftables.SetElement{setElement},
	)
	if err != nil {
		return
	}

	return
}

func (nft *NFTManager) initPreroutingChain(conn *nftables.Conn) (err error) {
	// Dispatch after destination NAT, including NAT hooks installed by Docker.
	// NAT's shared kernel hooks can retain an earlier registration's priority.
	// nft: type filter hook prerouting priority -99; policy accept;
	priority := nftables.ChainPriority(*nftables.ChainPriorityNATDest + 1)
	nft.preroutingChain = conn.AddChain(&nftables.Chain{
		Table:    nft.table,
		Name:     types.ChainPrerouting,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: &priority,
		Policy:   &nft.policy,
	})

	nft.addBypassRules(conn, nft.preroutingChain, true)

	// meta mark vmap @mark-vmap
	exprs := []expr.Any{
		&expr.Meta{
			Key:      expr.MetaKeyMARK,
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set mark-vmap dreg 0
			SourceRegister: 1,
			IsDestRegSet:   true,
			SetName:        nft.markTproxyMap.Name,
			SetID:          nft.markTproxyMap.ID,
		},
	}
	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: nft.table,
		Chain: nft.preroutingChain,
		Exprs: exprs,
	})

	return
}
