package routeman

import (
	"errors"
	"net"
	"os"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func (m *RouteManager) initializeNftableRuels() (err error) {
	defer Wrap(&err, "initializing nftable rules")

	for _, tp := range m.cfg.TProxies {
		err = m.nft.AddChainAndRulesForTProxy(tp)
		if err != nil {
			return
		}

		err = m.addRule(tp.Mark)
		if err != nil {
			return
		}
	}

	return
}

func (m *RouteManager) removeNftableRules() {
	err := m.nft.Clear()
	if err != nil {
		m.log.Errorw("Failed to delete nft table.",
			"error", err,
		)
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
	return
}

func (m *RouteManager) addRule(mark config.FireWallMark) (err error) {
	defer Wrap(&err, "add route rule")

	m.log.Infow("Adding route rule.",
		"mark", mark,
		"table", m.cfg.RouteTable,
	)

	// ip rule add fwmark <mark> lookup <table>

	rule := netlink.NewRule()
	rule.Family = netlink.FAMILY_ALL
	rule.Mark = int(mark) // WARN(black_desk): ???
	rule.Table = m.cfg.RouteTable

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
		if errors.Is(err, os.ErrExist) {
			m.log.Infow("Route already exists.",
				"route", route,
			)
			err = nil
		}
		if err != nil {
			return
		}

		m.route = append(m.route, route)
	}

	return
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

	err = m.nft.RemoveCgroups(paths)
	if err != nil {
		return
	}

	return
}
