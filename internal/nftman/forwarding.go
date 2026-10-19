// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

// SetRootTarget sets the shared fallback for local cgroups and forwarded traffic.
// A separate chain lets OUTPUT and PREROUTING use exactly the same decision.
func (nft *NFTManager) SetRootTarget(target types.Target) (err error) {
	defer Wrap(&err, "set root cgroup target")
	conn, err := nft.connector.Connect()
	if err != nil {
		return err
	}
	conn.FlushChain(nft.rootChain)
	verdict := &expr.Verdict{Kind: expr.VerdictReturn}
	switch target.Op {
	case types.TargetDrop:
		verdict.Kind = expr.VerdictDrop
	case types.TargetTProxy:
		verdict.Kind, verdict.Chain = expr.VerdictGoto, target.Chain
	}
	conn.AddRule(&nftables.Rule{Table: nft.table, Chain: nft.rootChain,
		Exprs: addDebugCounter([]expr.Any{verdict})})
	return conn.Flush()
}

func (nft *NFTManager) addRootFallback(conn *nftables.Conn, chain *nftables.Chain) {
	conn.AddRule(&nftables.Rule{Table: nft.table, Chain: chain,
		Exprs: addDebugCounter([]expr.Any{&expr.Verdict{Kind: expr.VerdictGoto, Chain: nft.rootChain.Name}})})
}

func (nft *NFTManager) addReturnRule(conn *nftables.Conn, chain *nftables.Chain, filters ...expr.Any) {
	filters = append(filters, &expr.Verdict{Kind: expr.VerdictReturn})
	conn.AddRule(&nftables.Rule{Table: nft.table, Chain: chain, Exprs: addDebugCounter(filters)})
}

func (nft *NFTManager) initForwardingChains(conn *nftables.Conn) error {
	// Conntrack has already run at priority -200. Classify before destination
	// NAT; TPROXY dispatch runs after NAT so the DNS mark can be cleared first.
	markPriority := nftables.ChainPriority(*nftables.ChainPriorityMangle - 2)
	chain := conn.AddChain(&nftables.Chain{
		Table: nft.table, Name: "forward-mangle", Type: nftables.ChainTypeFilter,
		Hooknum: nftables.ChainHookPrerouting, Priority: &markPriority, Policy: &nft.policy,
	})
	if err := nft.fillOutputMangleChain(conn, chain); err != nil {
		return err
	}
	// These lookups intentionally ignore the packet mark: the TPROXY policy
	// routing table itself has a local default route.
	for _, filter := range []struct {
		source bool
		op     expr.CmpOp
		typeID uint32
	}{
		{false, expr.CmpOpEq, unix.RTN_LOCAL},
		{false, expr.CmpOpNeq, unix.RTN_UNICAST},
		{true, expr.CmpOpEq, unix.RTN_LOCAL},
	} {
		nft.addReturnRule(conn, chain,
			&expr.Fib{Register: 1, ResultADDRTYPE: true,
				FlagSADDR: filter.source, FlagDADDR: !filter.source},
			&expr.Cmp{Op: filter.op, Register: 1, Data: binaryutil.NativeEndian.PutUint32(filter.typeID)},
		)
	}
	nft.addRootFallback(conn, chain)

	dnsChain := conn.AddChain(&nftables.Chain{
		Table: nft.table, Name: "prerouting-nat", Type: nftables.ChainTypeNAT,
		Hooknum: nftables.ChainHookPrerouting, Priority: nftables.ChainPriorityNATDest, Policy: &nft.policy,
	})
	conn.AddRule(&nftables.Rule{Table: nft.table, Chain: dnsChain,
		Exprs: addDebugCounter([]expr.Any{
			&expr.Meta{Key: expr.MetaKeyMARK, Register: 1},
			&expr.Lookup{SourceRegister: 1, IsDestRegSet: true,
				SetName: nft.markDNSMap.Name, SetID: nft.markDNSMap.ID},
		}),
	})
	return nil
}

// Excluded protocols must bypass marking as well as TPROXY dispatch.
func (nft *NFTManager) addProxyProtocolGuards(conn *nftables.Conn, chain *nftables.Chain, tp *config.TProxy) {
	if tp.NoIPv6 {
		nft.addReturnRule(conn, chain,
			&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
			&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV6}},
		)
	}
	if tp.NoUDP {
		if tp.DNSHijack != nil && !tp.NoIPv6 {
			// DNSHijack only supports IPv4; IPv6 UDP must still bypass.
			nft.addReturnRule(conn, chain,
				&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
				&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV6}},
				&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
				&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.IPPROTO_UDP}},
			)
		}
		filters := []expr.Any{
			&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
			&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.IPPROTO_UDP}},
		}
		if tp.DNSHijack != nil {
			filters = append(filters,
				&expr.Payload{DestRegister: 1, Base: expr.PayloadBaseTransportHeader, Offset: 2, Len: 2},
				&expr.Cmp{Op: expr.CmpOpNeq, Register: 1, Data: binaryutil.BigEndian.PutUint16(53)},
			)
		}
		nft.addReturnRule(conn, chain, filters...)
	}
}
