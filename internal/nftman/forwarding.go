// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

func (nft *NFTManager) initForwardingChains(conn *nftables.Conn) error {
	// Conntrack has already run at priority -200. Classify before destination
	// NAT; TPROXY dispatch runs after NAT so the DNS mark can be cleared first.
	markPriority := nftables.ChainPriority(*nftables.ChainPriorityMangle - 2)
	chain := conn.AddChain(&nftables.Chain{
		Table: nft.table, Name: types.ChainForwardMangle, Type: nftables.ChainTypeFilter,
		Hooknum: nftables.ChainHookPrerouting, Priority: &markPriority, Policy: &nft.policy,
	})
	if err := nft.addCommonFilters(conn, chain); err != nil {
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
	if nft.gateway {
		nft.addRootFallback(conn, chain)
	}

	dnsChain := conn.AddChain(&nftables.Chain{
		Table: nft.table, Name: types.ChainPreroutingNAT, Type: nftables.ChainTypeNAT,
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
