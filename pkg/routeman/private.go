// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"errors"
	"net"
	"os"
	"sort"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/vishvananda/netlink"
	"golang.org/x/exp/maps"
	"golang.org/x/sys/unix"
)

func (m *RouteManager) initializeNftableRuels() (err error) {
	defer Wrap(&err, "initializing nftable rules")

	err = m.nft.InitStructure()
	if err != nil {
		return
	}

	err = m.nft.AddChainAndRulesForTProxies(maps.Values(m.cfg.TProxies))
	if err != nil {
		return
	}

	err = m.nft.AddChainAndRulesForGateways(maps.Values(m.cfg.Gateways))
	if err != nil {
		return
	}

	for _, tp := range m.cfg.TProxies {
		err = m.addRule(tp.Mark)
		if err != nil {
			return
		}
	}

	for name := range m.cfg.Gateways {
		gw := m.cfg.Gateways[name]
		err = m.addGatewayRule(gw, m.gatewayTable[name])
		if err != nil {
			return
		}
	}

	return
}

func (m *RouteManager) removeNftableRules() {
	var err error

	err = m.nft.Clear()
	if err != nil {
		m.log.Errorw("Failed to delete nft table.",
			"error", err,
		)
		err = nil
	}

	for _, rule := range m.rule {
		err = netlink.RuleDel(rule)
		if err == nil {
			continue
		}

		m.log.Errorw("Failed to delete route rule.",
			"rule", rule,
			"error", err,
		)
	}

	err = m.nft.Release()
	if err != nil {
		m.log.Errorw("Failed to release NFTManager.",
			"error", err,
		)
		err = nil
	}

	return
}

func (m *RouteManager) addRuleWithFamily(
	mark config.FireWallMark, table, family int,
) (err error) {
	rule := netlink.NewRule()
	rule.Family = family
	rule.Mark = uint32(mark)
	rule.Table = table

	err = netlink.RuleAdd(rule)
	if errors.Is(err, os.ErrExist) {
		m.log.Infow("Rule already exists.")
		err = nil
	}
	if err != nil {
		return
	}

	m.rule = append(m.rule, rule)

	return
}

func (m *RouteManager) addRule(mark config.FireWallMark) (err error) {
	defer Wrap(&err, "add route rule")

	m.log.Infow("Adding route rule.",
		"mark", mark,
		"table", m.cfg.RouteTable,
	)

	// ip rule add fwmark <mark> lookup <table>
	err = m.addRuleWithFamily(mark, m.cfg.RouteTable, netlink.FAMILY_V4)
	if err != nil {
		return
	}

	// ip -6 rule add fwmark <mark> lookup <table>
	err = m.addRuleWithFamily(mark, m.cfg.RouteTable, netlink.FAMILY_V6)
	if err != nil {
		m.log.Errorw(
			"Failed to add ipv6 rule.",
			"mark", mark,
			"table", m.cfg.RouteTable,
			"error", err,
		)

		// TODO: It seems OK to not add this rule fail. But why?
		err = nil
	}

	return
}

func (m *RouteManager) addGatewayRule(
	gw *config.Gateway, table int,
) (err error) {
	defer Wrap(&err, "add route rule")

	m.log.Infow("Adding route rule.",
		"gateway", gw.Name,
		"mark", gw.Mark,
		"table", table,
	)

	// ip rule add fwmark <mark> lookup <table>
	err = m.addRuleWithFamily(gw.Mark, table, ipFamily(gw.IP))
	if err != nil {
		return
	}

	return
}

func ipFamily(ip string) int {
	if net.ParseIP(ip).To4() != nil {
		return netlink.FAMILY_V4
	}

	return netlink.FAMILY_V6
}

func (m *RouteManager) addRoute() (err error) {
	defer Wrap(&err, "add route")

	m.log.Infow("Adding route.",
		"table", m.cfg.RouteTable,
	)

	// ip route add local default dev lo table <table>

	var iface *net.Interface
	iface, err = net.InterfaceByName("lo")
	if err != nil {
		return
	}

	cidrStrs := []string{"0.0.0.0/0", "0::0/0"}

	for _, cidrStr := range cidrStrs {
		var cidr *net.IPNet

		_, cidr, err = net.ParseCIDR(cidrStr)
		if err != nil {
			return
		}

		route := &netlink.Route{
			LinkIndex: iface.Index,
			Scope:     unix.RT_SCOPE_HOST,
			Dst:       cidr,
			Table:     m.cfg.RouteTable,
			Type:      unix.RTN_LOCAL,
		}

		err = netlink.RouteAdd(route)
		if err != nil {
			return
		}

		m.route = append(m.route, route)
	}

	return
}

func (m *RouteManager) addGatewayRoutes() (err error) {
	defer Wrap(&err, "add gateway routes")

	names := maps.Keys(m.cfg.Gateways)
	sort.Strings(names)

	for _, name := range names {
		gw := m.cfg.Gateways[name]

		err = m.addGatewayRoute(gw, m.gatewayTable[name])
		if err != nil {
			return
		}
	}

	return
}

func (m *RouteManager) addGatewayRoute(
	gw *config.Gateway, table int,
) (err error) {
	defer Wrap(&err, "add route")

	m.log.Infow("Adding route.",
		"gateway", gw.Name,
		"table", table,
	)

	var iface *net.Interface
	iface, err = net.InterfaceByName(gw.Dev)
	if err != nil {
		return
	}

	var cidr *net.IPNet
	_, cidr, err = net.ParseCIDR(defaultCidr(gw.IP))
	if err != nil {
		return
	}

	route := &netlink.Route{
		LinkIndex: iface.Index,
		Dst:       cidr,
		Gw:        net.ParseIP(gw.IP),
		Table:     table,
		// Treat the gateway as directly reachable through Dev,
		// so no recursive lookup for the gateway is required.
		Flags: unix.RTNH_F_ONLINK,
	}

	err = netlink.RouteAdd(route)
	if err != nil {
		return
	}

	m.route = append(m.route, route)

	return
}

func defaultCidr(ip string) string {
	if net.ParseIP(ip).To4() != nil {
		return "0.0.0.0/0"
	}

	return "0::0/0"
}

func (m *RouteManager) removeRoute() {
	for i := range m.route {
		err := netlink.RouteDel(m.route[i])

		if err == nil {
			continue
		}

		m.log.Warnw("Failed to remove route",
			"error", err)
	}

	return
}

func (m *RouteManager) handleNewCgroups(paths []string) (err error) {
	defer Wrap(&err, "handle %d new cgroups", len(paths))

	routes := []types.Route{}

	for i := range paths {
		path := paths[i]

		m.log.Debugw("Checking route for cgroup.",
			"path", path,
		)

		var target types.Target
		for i := range m.matchers {
			if !m.matchers[i].reg.Match([]byte(path)) {
				continue
			}

			m.log.Debugw("Rule found for this cgroup",
				"cgroup", path,
				"rule", m.cfg.Rules[i].String(),
			)

			target = m.matchers[i].target

			break
		}

		if target.Op == types.TargetNoop {
			m.log.Debugw("No rule match this cgroup",
				"cgroup", path,
			)

			continue
		}

		routes = append(routes, types.Route{
			Path:   path,
			Target: target,
		})
	}

	err = m.nft.AddRoutes(routes)
	if err != nil {
		return
	}

	return
}

func (m *RouteManager) handleDeleteCgroups(paths []string) (err error) {
	defer Wrap(&err, "handle delete cgroup")

	m.log.Debugw("Handling delete cgroups.",
		"size", len(paths),
	)

	err = m.nft.RemoveRoutes(paths)
	if err != nil {
		return
	}

	return
}
