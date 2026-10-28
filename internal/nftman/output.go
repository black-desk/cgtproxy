// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"sort"
	"strings"

	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

func (nft *NFTManager) initOutputMangleChain(conn *nftables.Conn) (err error) {
	// type route hook output priority mangle; policy accept;
	nft.outputMangleChain = conn.AddChain(&nftables.Chain{
		Table:    nft.table,
		Name:     types.ChainOutputMangle,
		Type:     nftables.ChainTypeRoute,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &nft.policy,
	})

	return
}

func (nft *NFTManager) fillOutputCgroupRules(conn *nftables.Conn, elements map[string]nftables.SetElement) error {
	if err := nft.addCommonFilters(conn, nft.outputMangleChain); err != nil {
		return err
	}
	levelSet := map[int]struct{}{}
	for path := range elements {
		levelSet[strings.Count(path, "/")] = struct{}{}
	}
	levels := make([]int, 0, len(levelSet))
	for level := range levelSet {
		levels = append(levels, level)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(levels)))
	for _, level := range levels {
		if err := nft.addCgroupRuleForLevel(conn, level); err != nil {
			return err
		}
	}
	nft.addRootFallback(conn, nft.outputMangleChain)
	return nil
}

func (nft *NFTManager) initOutputNATChain(conn *nftables.Conn) (err error) {
	// type nat hook prerouting priority -100; policy accept;
	nft.outputNATChain = conn.AddChain(&nftables.Chain{
		Table:    nft.table,
		Name:     types.ChainOutputNAT,
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityNATDest,
		Policy:   &nft.policy,
	})

	// meta mark vmap @
	exprs := []expr.Any{
		&expr.Meta{
			Key:      expr.MetaKeyMARK,
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set mark-vmap dreg 0
			SourceRegister: 1,
			IsDestRegSet:   true,
			SetName:        nft.markDNSMap.Name,
			SetID:          nft.markDNSMap.ID,
		},
	}
	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: nft.table,
		Chain: nft.outputNATChain,
		Exprs: exprs,
	})

	return
}
