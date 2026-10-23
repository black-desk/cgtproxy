// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/vishvananda/netlink"
	"golang.org/x/exp/maps"
	"golang.org/x/sys/unix"
)

func (m *RouteManager) initializeNftableRules(paths []string) (err error) {
	defer Wrap(&err, "initializing nftable rules")

	for _, tp := range m.cfg.TProxies {
		err = m.addRule(tp)
		if err != nil {
			return
		}
	}

	return m.nft.Initialize(maps.Values(m.cfg.TProxies), m.targetForPath("/"), m.routesForPaths(paths))
}

func (m *RouteManager) removeNftableRules() {
	var err error

	err = m.nft.Clear()
	if err != nil {
		m.log.Errorf("Failed to delete the cgtproxy nft table: %v", err)
		err = nil
	}

	for _, rule := range m.rule {
		err = netlink.RuleDel(rule)
		if err == nil {
			continue
		}

		m.log.Errorf("Failed to delete policy rule (family=%d, mark=%#x, table=%d): %v", rule.Family, rule.Mark, rule.Table, err)
	}

	err = m.nft.Release()
	if err != nil {
		m.log.Errorf("Failed to release the nftables connection: %v", err)
		err = nil
	}

	return
}

func (m *RouteManager) addRuleWithFamily(
	mark config.FireWallMark, family int,
) (err error) {
	rule := netlink.NewRule()
	rule.Family = family
	rule.Mark = uint32(mark)
	rule.Table = m.cfg.RouteTable

	err = netlink.RuleAdd(rule)
	if errors.Is(err, os.ErrExist) {
		m.log.Infof("Policy rule already exists (family=%d, mark=%#x, table=%d).", family, mark, rule.Table)
		err = nil
	}
	if err != nil {
		return
	}

	m.rule = append(m.rule, rule)

	return
}

func (m *RouteManager) addRule(tp *config.TProxy) (err error) {
	defer Wrap(&err, "add route rule")
	mark := tp.Mark

	m.log.Infof("Adding policy rules for proxy %q: mark=%#x, table=%d, IPv6=%t.", tp.Name, mark, m.cfg.RouteTable, !tp.NoIPv6)

	// ip rule add fwmark <mark> lookup <table>
	err = m.addRuleWithFamily(mark, netlink.FAMILY_V4)
	if err != nil {
		return
	}

	if tp.NoIPv6 {
		return nil
	}

	// ip -6 rule add fwmark <mark> lookup <table>
	err = m.addRuleWithFamily(mark, netlink.FAMILY_V6)
	if err != nil {
		m.log.Errorf("Failed to add IPv6 policy rule (mark=%#x, table=%d): %v", mark, m.cfg.RouteTable, err)
	}

	return
}

func (m *RouteManager) addRoute() (err error) {
	defer Wrap(&err, "add route")

	// ip route add local default dev lo table <table>

	var iface *net.Interface
	iface, err = net.InterfaceByName("lo")
	if err != nil {
		return
	}

	cidrStrs := []string{"0.0.0.0/0"}
	for _, tp := range m.cfg.TProxies {
		if !tp.NoIPv6 {
			cidrStrs = append(cidrStrs, "::/0")
			break
		}
	}

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
		m.log.Infof("Added local route: destination=%s, device=lo, table=%d.", cidr, route.Table)
	}

	return
}

func (m *RouteManager) removeRoute() {
	for i := range m.route {
		err := netlink.RouteDel(m.route[i])

		if err == nil {
			continue
		}

		m.log.Warnf("Failed to remove route %s from table %d: %v", m.route[i].Dst, m.route[i].Table, err)
	}

	return
}

func (m *RouteManager) routesForPaths(paths []string) []types.Route {
	routes := []types.Route{}
	root := filepath.Clean(string(m.cfg.CgroupRoot))

	for i := range paths {
		path := paths[i]

		m.log.Debugf("Checking routing configuration for cgroup %q.", path)

		// Match cgroup paths relative to cgroupfs, retaining the filesystem
		// path in the route for stat(2) and deletion handling.
		matchPath := strings.TrimPrefix(filepath.Clean(path), root)
		matchPath = "/" + strings.TrimPrefix(matchPath, "/")
		target := m.targetForPath(matchPath)

		if target.Op == types.TargetNoop {
			m.log.Debugf("No routing rule matches cgroup %q; inheriting the ancestor or root target.", matchPath)

			continue
		}

		routes = append(routes, types.Route{
			Path:   path,
			Target: target,
		})
	}

	return routes
}

func (m *RouteManager) targetForPath(path string) types.Target {
	for i := range m.matchers {
		if m.matchers[i].reg.MatchString(path) {
			m.log.Debugf("Cgroup %q matched rules[%d]: %s.", path, i, &m.cfg.Rules[i])
			return m.matchers[i].target
		}
	}
	return types.Target{}
}
