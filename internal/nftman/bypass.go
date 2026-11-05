// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

func (nft *NFTManager) initIPV4BypassSet(conn *nftables.Conn) (err error) {
	defer Wrap(&err, "prepare ipv4 bypass set")

	nft.ipv4BypassSet = &nftables.Set{
		Table:        nft.table,
		Name:         "bypass",
		KeyType:      nftables.TypeIPAddr,
		KeyByteOrder: binaryutil.BigEndian,
		Interval:     true,
	}

	err = conn.AddSet(nft.ipv4BypassSet, nil)
	if err != nil {
		return
	}
	err = queueSetElements(nft.ipv4BypassSet, bypassSetElements(nft.bypassIPv4), conn.SetAddElements)

	return
}

func (nft *NFTManager) initIPV6BypassSet(conn *nftables.Conn) (err error) {
	nft.ipv6BypassSet = &nftables.Set{
		Table:        nft.table,
		Name:         "bypass6",
		KeyType:      nftables.TypeIP6Addr,
		KeyByteOrder: binaryutil.BigEndian,
		Interval:     true,
	}

	err = conn.AddSet(nft.ipv6BypassSet, nil)
	if err != nil {
		return
	}
	err = queueSetElements(nft.ipv6BypassSet, bypassSetElements(nft.bypassIPv6), conn.SetAddElements)

	return
}

func (nft *NFTManager) initProxyMarkSet(conn *nftables.Conn) error {
	nft.proxyMarkSet = &nftables.Set{
		Table: nft.table, Name: "proxy-marks", KeyType: nftables.TypeMark,
	}
	return conn.AddSet(nft.proxyMarkSet, nil)
}

func (nft *NFTManager) initOutputBypassChain(conn *nftables.Conn) {
	priority := nftables.ChainPriority(*nftables.ChainPriorityNATDest + 1)
	chain := conn.AddChain(&nftables.Chain{
		Table: nft.table, Name: types.ChainOutputBypass, Type: nftables.ChainTypeRoute,
		Hooknum: nftables.ChainHookOutput, Priority: &priority, Policy: &nft.policy,
	})
	// A route chain triggers a new lookup when clearing a mark after OUTPUT
	// DNAT. Otherwise the packet can retain the earlier local TPROXY route.
	nft.addBypassRules(conn, chain, true)
}

func (nft *NFTManager) addBypassRules(conn *nftables.Conn, chain *nftables.Chain, clearProxyMark bool) {
	for _, family := range []struct {
		protocol byte
		offset   uint32
		length   uint32
		set      *nftables.Set
	}{
		{unix.NFPROTO_IPV4, 16, 4, nft.ipv4BypassSet},
		{unix.NFPROTO_IPV6, 24, 16, nft.ipv6BypassSet},
	} {
		exprs := []expr.Any{
			&expr.Meta{Key: expr.MetaKeyNFPROTO, Register: 1},
			&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{family.protocol}},
			&expr.Payload{DestRegister: 1, Base: expr.PayloadBaseNetworkHeader,
				Offset: family.offset, Len: family.length},
			&expr.Lookup{SourceRegister: 1, SetID: family.set.ID, SetName: family.set.Name},
		}
		if clearProxyMark {
			// Marks are reserved by their exact configured values. Leave marks
			// used by other network components untouched.
			exprs = append(exprs,
				&expr.Meta{Key: expr.MetaKeyMARK, Register: 1},
				&expr.Lookup{SourceRegister: 1, SetID: nft.proxyMarkSet.ID, SetName: nft.proxyMarkSet.Name},
				&expr.Immediate{Register: 1, Data: binaryutil.NativeEndian.PutUint32(0)},
				&expr.Meta{Key: expr.MetaKeyMARK, Register: 1, SourceRegister: true},
			)
		}
		exprs = append(exprs, &expr.Verdict{Kind: expr.VerdictReturn})
		conn.AddRule(&nftables.Rule{Table: nft.table, Chain: chain, Exprs: addDebugCounter(exprs)})
	}
}
