// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"errors"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
)

// Initialize installs the complete ruleset, including the first cgroup scan,
// in one transaction. Policy routes must be ready before calling it.
func (nft *NFTManager) Initialize(tps []*config.TProxy, rootTarget types.Target, routes []types.Route) (err error) {
	defer Wrap(&err, "initialize nftable rules")

	conn, err := nft.connector.Connect()
	if err != nil {
		return err
	}
	defer nft.discardOnError(&err)

	// Publish state only after the kernel accepts the complete transaction.
	// In particular, cleanup must not delete a table we failed to create.
	next := *nft
	if err = next.initStructure(conn); err != nil {
		return err
	}
	if err = next.addChainAndRulesForTProxies(conn, tps); err != nil {
		return err
	}
	next.setRootTarget(conn, rootTarget)
	next.cgroupMapElement, err = next.cgroupElementsForRoutes(routes)
	if err != nil {
		return err
	}
	if len(next.cgroupMapElement) != 0 {
		elements := make([]nftables.SetElement, 0, len(next.cgroupMapElement))
		for _, element := range next.cgroupMapElement {
			elements = append(elements, element)
		}
		if err = queueCgroupElements(next.cgroupMap, elements, conn.SetAddElements); err != nil {
			return err
		}
	}
	if err = next.fillOutputCgroupRules(conn, next.cgroupMapElement); err != nil {
		return err
	}
	if err = conn.Flush(); err != nil {
		return err
	}
	*nft = next
	nft.log.Info("Initial nft rules committed.")
	nft.dumpNFTableRules()
	return nil
}

// Releasing the connection also discards commands buffered before a build error.
// Otherwise a later operation, including shutdown cleanup, could submit them.
func (nft *NFTManager) discardOnError(err *error) {
	if *err != nil {
		*err = errors.Join(*err, nft.connector.Release())
	}
}

func (nft *NFTManager) addChainAndRulesForTProxies(conn *nftables.Conn, tps []*config.TProxy) (err error) {
	if len(tps) == 0 {
		return
	}

	defer Wrap(
		&err,
		"add chain and rules to nft table for tproxies %#v",
		tps,
	)

	for _, tp := range tps {
		nft.log.Debugw("Generating chain and rules for tproxy.",
			"tproxy", tp,
		)

		_, err = nft.addMarkChainForTProxy(conn, tp)
		if err != nil {
			return
		}

		var chain *nftables.Chain

		chain, err = nft.addTproxyChainForTProxy(conn, tp)
		if err != nil {
			return
		}

		err = nft.updateMarkTproxyMap(conn, tp.Mark, chain.Name)
		if err != nil {
			return
		}

		if tp.DNSHijack != nil {
			chain, err = nft.addDNSChainForTproxy(conn, tp)
			if err != nil {
				return
			}

			err = nft.updateMarkDNSMap(conn, tp.Mark, chain.Name)
			if err != nil {
				return
			}
		}

		nft.log.Debug("Chain and rules generated for this tproxy.",
			"tproxy", tp,
		)
	}

	return
}

func (nft *NFTManager) initStructure(conn *nftables.Conn) (err error) {
	defer Wrap(&err, "build initial structure")

	nft.log.Debug("Building nft table structure.")

	nft.table = conn.CreateTable(&nftables.Table{
		Name:   NftTableName,
		Family: nftables.TableFamilyINet,
	})

	err = nft.initIPV4BypassSet(conn)
	if err != nil {
		return
	}

	err = nft.initIPV6BypassSet(conn)
	if err != nil {
		return
	}

	nft.initProtoSet()

	err = nft.initCgroupMap(conn)
	if err != nil {
		return
	}

	err = nft.initMarkMap(conn)
	if err != nil {
		return
	}

	err = nft.initMarkDNSMap(conn)
	if err != nil {
		return
	}

	nft.policy = nftables.ChainPolicyAccept
	nft.rootChain = conn.AddChain(&nftables.Chain{Table: nft.table, Name: "cgroup-root"})

	err = nft.initOutputMangleChain(conn)
	if err != nil {
		return
	}

	err = nft.initOutputNATChain(conn)
	if err != nil {
		return
	}

	err = nft.initPreroutingChain(conn)
	if err != nil {
		return
	}

	err = nft.initForwardingChains(conn)
	if err != nil {
		return
	}

	return
}
