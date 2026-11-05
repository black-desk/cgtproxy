// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"net"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

func (t *NFTManager) addDNSChainForTproxy(
	conn *nftables.Conn, tp *config.TProxy,
) (
	ret *nftables.Chain, err error,
) {
	chain := &nftables.Chain{
		Table: t.table,
		Name:  tp.Name + "-DNS",
	}

	conn.AddChain(chain)

	defer func() {
		if err != nil {
			return
		}

		ret = chain
	}()

	exprs := []expr.Any{
		&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV4}},
		&expr.Meta{ // meta load l4proto => reg 1
			Key:      expr.MetaKeyL4PROTO,
			Register: 1,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00000011
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{unix.IPPROTO_UDP},
		},
		&expr.Payload{ // payload load 2b @ transport header + 2 => reg 1
			OperationType: expr.PayloadLoad,
			DestRegister:  1,
			Base:          expr.PayloadBaseTransportHeader,
			Offset:        2,
			Len:           2,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00003500
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     binaryutil.BigEndian.PutUint16(53),
		},
		// DNAT chooses the DNS destination; do not route it back into TPROXY.
		&expr.Immediate{Register: 1, Data: binaryutil.NativeEndian.PutUint32(0)},
		&expr.Meta{Key: expr.MetaKeyMARK, SourceRegister: true, Register: 1},
		&expr.Immediate{ // immediate reg 1 xxx
			Register: 1,
			Data:     net.ParseIP(*tp.DNSHijack.IP).To4(),
		},
		&expr.Immediate{ // immediate reg 2 xxx
			Register: 2,
			Data:     binaryutil.BigEndian.PutUint16(tp.DNSHijack.Port),
		},
		&expr.NAT{ // nat dnat ip addr_min reg 1
			Type:        expr.NATTypeDestNAT,
			Family:      unix.NFPROTO_IPV4,
			RegAddrMin:  1,
			RegProtoMin: 2,
		},
	}
	rule := &nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	}

	conn.AddRule(rule)

	if !tp.DNSHijack.TCP {
		return
	}

	exprs = []expr.Any{
		&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV4}},
		&expr.Meta{ // meta load l4proto => reg 1
			Key:      expr.MetaKeyL4PROTO,
			Register: 1,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00000006
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{unix.IPPROTO_TCP},
		},
		&expr.Payload{ // payload load 2b @ transport header + 2 => reg 1
			OperationType:  expr.PayloadLoad,
			DestRegister:   1,
			SourceRegister: 0,
			Base:           expr.PayloadBaseTransportHeader,
			Offset:         2,
			Len:            2,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00003500
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     binaryutil.BigEndian.PutUint16(53),
		},
		&expr.Immediate{Register: 1, Data: binaryutil.NativeEndian.PutUint32(0)},
		&expr.Meta{Key: expr.MetaKeyMARK, SourceRegister: true, Register: 1},
		&expr.Immediate{ // immediate reg 1 xxx
			Register: 1,
			Data:     net.ParseIP(*tp.DNSHijack.IP).To4(),
		},
		&expr.Immediate{ // immediate reg 2 xxx
			Register: 2,
			Data:     binaryutil.BigEndian.PutUint16(tp.DNSHijack.Port),
		},
		&expr.NAT{ // nat dnat ip addr_min reg 1
			Type:        expr.NATTypeDestNAT,
			Family:      unix.NFPROTO_IPV4,
			RegAddrMin:  1,
			RegProtoMin: 2,
		},
	}
	rule = &nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	}

	conn.AddRule(rule)

	return
}

func (nft *NFTManager) addDNSHijackGuard(conn *nftables.Conn, chain *nftables.Chain, tp *config.TProxy) {
	if tp.DNSHijack == nil {
		return
	}
	// NAT chains only run on the first packet. Skip marking subsequent DNS
	// packets only when the original port and translated resolver match this
	// proxy's DNS hijack. Other DNAT connections must still be marked on every
	// packet. Reading the conntrack tuples also avoids reserving a ct mark.
	filters := []expr.Any{
		&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.NFPROTO_IPV4}},
		&expr.Ct{Register: 1, Key: expr.CtKeySTATUS},
		&expr.Bitwise{SourceRegister: 1, DestRegister: 1, Len: 4,
			Mask: binaryutil.NativeEndian.PutUint32(1 << 5), // IPS_DST_NAT
			Xor:  binaryutil.NativeEndian.PutUint32(0)},
		&expr.Cmp{Op: expr.CmpOpNeq, Register: 1, Data: binaryutil.NativeEndian.PutUint32(0)},
		&expr.Ct{Register: 1, Key: expr.CtKeyPROTODST},
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: binaryutil.BigEndian.PutUint16(53)},
		&expr.Ct{Register: 1, Key: expr.CtKeySRC, Direction: ctReplyDirection}, // reply source is the DNAT destination
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: net.ParseIP(*tp.DNSHijack.IP).To4()},
		&expr.Ct{Register: 1, Key: expr.CtKeyPROTOSRC, Direction: ctReplyDirection},
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: binaryutil.BigEndian.PutUint16(tp.DNSHijack.Port)},
	}
	if !tp.DNSHijack.TCP {
		filters = append(filters,
			&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
			&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{unix.IPPROTO_UDP}},
		)
	}
	nft.addReturnRule(conn, chain, filters...)
}

func (nft *NFTManager) initMarkDNSMap(conn *nftables.Conn) (err error) {
	nft.markDNSMap = &nftables.Set{
		Table:        nft.table,
		Name:         "mark-dns-vmap",
		KeyType:      nftables.TypeMark,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	err = conn.AddSet(nft.markDNSMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	return
}

func (t *NFTManager) updateMarkDNSMap(
	conn *nftables.Conn, mark config.FireWallMark, chain string,
) (
	err error,
) {
	setElement := nftables.SetElement{
		Key: binaryutil.NativeEndian.PutUint32(uint32(mark)),
		VerdictData: &expr.Verdict{
			Kind:  expr.VerdictGoto,
			Chain: chain,
		},
	}
	err = conn.SetAddElements(
		t.markDNSMap,
		[]nftables.SetElement{setElement},
	)
	if err != nil {
		return
	}

	return
}
