// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"regexp"
	"sort"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

type RouteManager struct {
	cgroupEventsChan <-chan types.CGroupEvents

	nft interfaces.NFTManager
	cfg *config.Config
	log *zap.SugaredLogger

	matchers []*struct {
		reg    *regexp.Regexp
		target types.Target
	}

	// gatewayTable stores the route table assigned to each gateway.
	// Tables are allocated from RouteTable + 1 in gateway name order,
	// so that multiple gateways never share one table.
	gatewayTable map[string]int

	rule  []*netlink.Rule
	route []*netlink.Route
}

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/pkg/routeman.RouteManager -as interfaces.RouteManager -o ../interfaces/routeman.go

func New(opts ...Opt) (ret *RouteManager, err error) {
	defer Wrap(&err, "create the nftable rule manager")

	m := &RouteManager{}
	for i := range opts {
		m, err = opts[i](m)
		if err != nil {
			return
		}
	}

	if m.log == nil {
		m.log = zap.NewNop().Sugar()
	}

	for i := range m.cfg.Rules {
		regex := m.cfg.Rules[i].Match
		var matcher struct {
			reg    *regexp.Regexp
			target types.Target
		}

		matcher.reg, err = regexp.Compile(regex)
		if err != nil {
			return
		}

		if m.cfg.Rules[i].Direct {
			matcher.target.Op = types.TargetDirect
		} else if m.cfg.Rules[i].Drop {
			matcher.target.Op = types.TargetDrop
		} else if m.cfg.Rules[i].TProxy != "" {
			matcher.target.Op = types.TargetTProxy
			matcher.target.Chain =
				m.cfg.TProxies[m.cfg.Rules[i].TProxy].Name
		} else if m.cfg.Rules[i].Gateway != "" {
			matcher.target.Op = types.TargetGateway
			matcher.target.Chain =
				m.cfg.Gateways[m.cfg.Rules[i].Gateway].Name
		} else {
			panic("this should never happened.")
		}

		if matcher.target.Chain != "" {
			matcher.target.Chain += "-MARK"
		}

		m.matchers = append(m.matchers, &matcher)
	}

	m.gatewayTable = map[string]int{}
	names := maps.Keys(m.cfg.Gateways)
	sort.Strings(names)
	for i, name := range names {
		m.gatewayTable[name] = m.cfg.RouteTable + i + 1
	}

	ret = m

	m.log.Debugw("Create a new route manager.")
	return
}

type Opt func(m *RouteManager) (ret *RouteManager, err error)

func WithNFTMan(t interfaces.NFTManager) Opt {
	return func(m *RouteManager) (ret *RouteManager, err error) {
		if t == nil {
			err = ErrNFTManagerMissing
			return
		}

		m.nft = t
		ret = m
		return
	}
}

func WithConfig(c *config.Config) Opt {
	return func(m *RouteManager) (ret *RouteManager, err error) {
		if c == nil {
			err = ErrConfigMissing
			return
		}

		m.cfg = c
		ret = m
		return
	}
}

func WithCGroupEventChan(ch <-chan types.CGroupEvents) Opt {
	return func(m *RouteManager) (ret *RouteManager, err error) {
		if ch == nil {
			err = ErrCGroupEventChanMissing
			return
		}

		m.cgroupEventsChan = ch
		ret = m
		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(m *RouteManager) (ret *RouteManager, err error) {
		m.log = log
		ret = m
		return
	}
}
