package rulemanager

import (
	"net"

	"github.com/black-desk/deepin-network-proxy-manager/internal/core/table"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func (m *RuleManager) Run() (err error) {
	defer Wrap(&err, "Error occurs while running the nftable rules manager.")

	defer m.removeNftableRules()
	err = m.initializeNftableRuels()
	if err != nil {
		return
	}

	defer m.removeRoute()
	err = m.addRoute()
	if err != nil {
		return
	}

	defer m.removeRule()
	err = m.addRule()
	if err != nil {
		return
	}

	for event := range m.cgroupEventChan {
		switch event.EventType {
		case types.CgroupEventTypeNew:
			m.handleNewCgroup(event.Path)
		case types.CgroupEventTypeDelete:
			m.handleDeleteCgroup(event.Path)
		}
	}
	return
}

func (m *RuleManager) initializeNftableRuels() (err error) {
	defer Wrap(&err, "Failed to initialize nftable ruels.")

	for _, tp := range m.cfg.TProxies {
		err = m.nft.AddChainAndRulesForTProxy(tp)
		if err != nil {
			return
		}
	}

	return
}

func (m *RuleManager) removeNftableRules() (err error) {
	err = m.nft.Clear()
	return
}

func (m *RuleManager) addRoute() (err error) {
	defer Wrap(&err, "Failed to add route.")

	var iface *net.Interface
	iface, err = net.InterfaceByName("lo")
	if err != nil {
		return
	}

	route := &netlink.Route{
		LinkIndex: iface.Index,
		Scope:     unix.RT_SCOPE_HOST,
		Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
		Protocol:  unix.RTPROT_BOOT,
		Table:     m.cfg.RouteTable,
		Type:      unix.RTN_LOCAL,
	}

	err = netlink.RouteAdd(route)
	if err != nil {
		return
	}

	m.route = route

	return
}

func (m *RuleManager) removeRoute() {
	if m.route == nil {
		return
	}

	err := netlink.RouteDel(m.route)

	if err == nil {
		return
	}

	Log.Warnw("Failed to delete route", "error", err)

	return
}

func (m *RuleManager) addRule() (err error) {
	defer Wrap(&err, "Failed to add route rule.")

	rule := netlink.NewRule()
	rule.Family = netlink.FAMILY_ALL
	rule.Mark = int(m.cfg.Mark) // WARN(black_desk): ???
	rule.Table = m.cfg.RouteTable

	err = netlink.RuleAdd(rule)
	if err != nil {
		return
	}

	m.rule = rule

	return
}

func (m *RuleManager) removeRule() {
	if m.rule == nil {
		return
	}

	err := netlink.RuleDel(m.rule)

	if err == nil {
		return
	}

	Log.Warnw("Failed to delete rule", "error", err)

	return
}

func (m *RuleManager) handleNewCgroup(path string) {
	var target table.Target
	for i := range m.matchers {
		if !m.matchers[i].reg.Match([]byte(path)) {
			continue
		}

		Log.Infow("Rule found for this cgroup",
			"cgroup", path,
			"rule", m.cfg.Rules[i].String(),
		)

		target = m.matchers[i].target

		break
	}

	if target.Op == table.TargetNoop {
		Log.Infow("No rule match this cgroup",
			"cgroup", path,
		)
		return
	}

	err := m.nft.AddCgroup(path, &target)
	if err != nil {
		Log.Errorw("Failed to update nft for new cgroup",
			"error", err,
		)
	}
}

func (m *RuleManager) handleDeleteCgroup(path string) {
	err := m.nft.RemoveCgroup(path)
	if err != nil {
		Log.Errorw("Failed to update nft for removed cgroup", "error", err)
	}
}
