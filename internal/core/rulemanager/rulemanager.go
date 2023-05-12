package rulemanager

import (
	"fmt"
	"net"
	"regexp"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/monitor"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/rulemanager/table"
	"github.com/black-desk/deepin-network-proxy-manager/internal/inject"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type RuleManager struct {
	EventChan <-chan *monitor.CgroupEvent `inject:"true"`

	Nft *table.Table   `inject:"true"`
	Cfg *config.Config `inject:"true"`

	matchers []*regexp.Regexp

	rule  *netlink.Rule
	route *netlink.Route
}

func New(container *inject.Container) (m *RuleManager, err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Catch()+
			"Failed to create the nftable rule manager:\n%w",
			err,
		)
	}()

	m = &RuleManager{}
	err = container.Fill(m)
	if err != nil {
		return
	}

	for i := range m.Cfg.Rules {
		regex := m.Cfg.Rules[i].Match
		var matcher *regexp.Regexp
		matcher, err = regexp.Compile(regex)
		if err != nil {
			return
		}

		m.matchers = append(m.matchers, matcher)
	}

	return
}

func (m *RuleManager) Run() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Catch()+
			"Error occurs while running the nftable rules manager:\n%w",
			err,
		)
	}()

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

	for event := range m.EventChan {
		switch event.EventType {
		case monitor.CgroupEventTypeNew:
			m.handleNewCgroup(event.Path)
		case monitor.CgroupEventTypeDelete:
			m.handleDeleteCgroup(event.Path)
		}
	}
	return
}

func (m *RuleManager) initializeNftableRuels() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Catch()+
			"Failed to initialize nftable ruels:\n%w",
			err,
		)
	}()

	for _, tp := range m.Cfg.TProxies {
		// NOTE(black_desk): Same as `addChainForProxy`.
		_ = m.Nft.AddChainAndRulesForTProxy(tp)
	}

	err = m.Nft.FlushInitialContent()
	return
}

func (m *RuleManager) removeNftableRules() (err error) {
	err = m.Nft.Clear()
	return
}

func (m *RuleManager) addRoute() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Catch()+
			"Failed to add route:\n%w",
			err,
		)
	}()

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
		Table:     m.Cfg.RouteTable,
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
	err := netlink.RouteDel(m.route)

	if err == nil {
		return
	}

	log.Warning().Printf("failed to delete route: %s", err.Error())

	return
}

func (m *RuleManager) addRule() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Catch()+
			"Failed to add route rule:\n%w",
			err,
		)
	}()

	rule := netlink.NewRule()
	rule.Family = netlink.FAMILY_ALL
	rule.Mark = int(m.Cfg.Mark) // WARN(black_desk): ???
	rule.Table = m.Cfg.RouteTable

	err = netlink.RuleAdd(rule)
	if err != nil {
		return
	}

	m.rule = rule

	return
}

func (m *RuleManager) removeRule() {
	err := netlink.RuleDel(m.rule)

	if err == nil {
		return
	}

	log.Warning().Printf("failed to delete rule: %s", err.Error())

	return
}

func (m *RuleManager) handleNewCgroup(path string) {
	var target table.Target
	for i := range m.matchers {
		if !m.matchers[i].Match([]byte(path)) {
			continue
		}

		if m.Cfg.Rules[i].Direct {
			target.Op = table.TargetDirect
		} else if m.Cfg.Rules[i].Drop {
			target.Op = table.TargetDrop
		} else if m.Cfg.Rules[i].Proxy != "" {
			target.Op = table.TargetTProxy
			target.Chain = m.Cfg.Proxies[m.Cfg.Rules[i].Proxy].TProxy.Name
		} else if m.Cfg.Rules[i].TProxy != "" {
			target.Op = table.TargetTProxy
			target.Chain = m.Cfg.TProxies[m.Cfg.Rules[i].TProxy].Name
		} else {
			panic("this should never happened.")
		}

		break
	}

	err := m.Nft.AddCgroup(path, &target)
	if err != nil {
		log.Err().Print(err.Error())
	}
}

func (m *RuleManager) handleDeleteCgroup(path string) {
	err := m.Nft.RemoveCgroup(path)
	if err != nil {
		log.Err().Printf(err.Error())
	}
}
