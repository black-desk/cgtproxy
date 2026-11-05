// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"errors"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
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
		if err = queueSetElements(next.cgroupMap, elements, conn.SetAddElements); err != nil {
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
	nft.log.Infof("Initial nft rules committed: table=%q, cgroup routes=%d, root target=%s, root chain=%q.", NftTableName, len(nft.cgroupMapElement), rootTarget.Op, rootTarget.Chain)
	paths := make([]string, 0, len(nft.cgroupMapElement))
	for path := range nft.cgroupMapElement {
		paths = append(paths, path)
	}
	nft.logCgroupRoutes("Added", nft.cgroupMapElement, paths)
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
		nft.log.Debugf("Generating nft chains and rules for proxy %q: %+v.", tp.Name, tp)
		if err = conn.SetAddElements(nft.proxyMarkSet, []nftables.SetElement{
			{Key: binaryutil.NativeEndian.PutUint32(uint32(tp.Mark))},
		}); err != nil {
			return err
		}

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

		nft.log.Debugf("Generated nft chains and rules for proxy %q.", tp.Name)
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
	err = nft.initProxyMarkSet(conn)
	if err != nil {
		return
	}

	nft.policy = nftables.ChainPolicyAccept
	nft.rootChain = conn.AddChain(&nftables.Chain{Table: nft.table, Name: types.ChainCgroupRoot})

	err = nft.initOutputMangleChain(conn)
	if err != nil {
		return
	}

	err = nft.initOutputNATChain(conn)
	if err != nil {
		return
	}
	nft.initOutputBypassChain(conn)

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

// setRootTarget sets the shared fallback for local and forwarded traffic.
func (nft *NFTManager) setRootTarget(conn *nftables.Conn, target types.Target) {
	verdict := &expr.Verdict{Kind: expr.VerdictReturn}
	switch target.Op {
	case types.TargetTProxy:
		verdict.Kind, verdict.Chain = expr.VerdictGoto, target.Chain
	}
	conn.AddRule(&nftables.Rule{Table: nft.table, Chain: nft.rootChain,
		Exprs: addDebugCounter([]expr.Any{verdict})})
}

func (nft *NFTManager) addRootFallback(conn *nftables.Conn, chain *nftables.Chain) {
	conn.AddRule(&nftables.Rule{Table: nft.table, Chain: chain,
		Exprs: addDebugCounter([]expr.Any{&expr.Verdict{Kind: expr.VerdictGoto, Chain: nft.rootChain.Name}})})
}
