// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"errors"
	"fmt"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"golang.org/x/exp/maps"
)

// initializeIfNeeded queues base rules unless the table is already committed.
func (nft *NFTManager) initializeIfNeeded() (err error) {
	defer errwrap.Wrap(&err, "initialize nftable rules")

	if nft.initialized {
		return nil
	}

	conn, err := nft.connector.Connect()
	if err != nil {
		return err
	}

	// Build state separately so a preparation failure leaves no partial state.
	// Clear must not delete this table until UpdateRoutes commits its creation.
	next := *nft
	if err = next.initStructure(conn); err != nil {
		return err
	}
	if err = next.addChainAndRulesForTProxies(conn, maps.Values(nft.tproxies)); err != nil {
		return err
	}
	var gatewayTarget types.Target
	if nft.gateway != nil {
		gatewayTarget.Op = types.TargetDirect
		if !nft.gateway.Direct {
			gatewayTarget.Op = types.TargetTProxy
			gatewayTarget.Chain = nft.tproxies[nft.gateway.TProxy].Name + "-MARK"
		}
	}
	if err = next.initForwardingChains(conn, gatewayTarget); err != nil {
		return err
	}
	*nft = next
	nft.log.Debugf("Base nft rules prepared: table=%q, gateway target=%s, gateway chain=%q.", NftTableName, gatewayTarget.Op, gatewayTarget.Chain)
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

	for _, tp := range tps {
		nft.log.Debugf("Generating nft chains and rules for proxy %q: %+v.", tp.Name, tp)
		// nft: add element inet cgtproxy proxy-marks { <tp.Mark> }
		if err = conn.SetAddElements(nft.proxyMarkSet, []nftables.SetElement{
			{Key: binaryutil.NativeEndian.PutUint32(uint32(tp.Mark))},
		}); err != nil {
			return fmt.Errorf("register mark for proxy %q: %w", tp.Name, err)
		}

		_, err = nft.addMarkChainForTProxy(conn, tp)
		if err != nil {
			return
		}

		var chain *nftables.Chain

		chain, err = nft.addTproxyChainForTProxy(conn, tp)
		if err != nil {
			return fmt.Errorf("build TPROXY chain for proxy %q: %w", tp.Name, err)
		}

		err = nft.updateMarkTproxyMap(conn, tp.Mark, chain.Name)
		if err != nil {
			return fmt.Errorf("register TPROXY target for proxy %q: %w", tp.Name, err)
		}

		if tp.DNSHijack != nil {
			chain, err = nft.addDNSChainForTproxy(conn, tp)
			if err != nil {
				return
			}

			err = nft.updateMarkDNSMap(conn, tp.Mark, chain.Name)
			if err != nil {
				return fmt.Errorf("register DNS target for proxy %q: %w", tp.Name, err)
			}
		}

		nft.log.Debugf("Generated nft chains and rules for proxy %q.", tp.Name)
	}

	return
}

func (nft *NFTManager) initStructure(conn *nftables.Conn) (err error) {
	nft.log.Debug("Building nft table structure.")

	// Bind the table to the lasting netlink connection so the kernel also
	// removes it if the process exits without running shutdown cleanup.
	nft.table = conn.CreateTable(&nftables.Table{
		Name:   NftTableName,
		Family: nftables.TableFamilyINet,
		Flags:  nftables.TableFlagOwner,
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

	return
}
