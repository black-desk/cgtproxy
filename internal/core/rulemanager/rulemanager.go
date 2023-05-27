package rulemanager

import (
	"regexp"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/table"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/vishvananda/netlink"
)

type RuleManager struct {
	cgroupEventChan <-chan *types.CgroupEvent `inject:"true"`

	nft *table.Table   `inject:"true"`
	cfg *config.Config `inject:"true"`

	matchers []*regexp.Regexp

	rule  *netlink.Rule
	route *netlink.Route
}

func New(opts ...Opt) (ret *RuleManager, err error) {
	defer Wrap(&err, "Failed to create the nftable rule manager.")

	m := &RuleManager{}
	for i := range opts {
		m, err = opts[i](m)
		if err != nil {
			return
		}
	}

	for i := range m.cfg.Rules {
		regex := m.cfg.Rules[i].Match
		var matcher *regexp.Regexp
		matcher, err = regexp.Compile(regex)
		if err != nil {
			return
		}

		m.matchers = append(m.matchers, matcher)
	}

	ret = m
	return
}

type Opt func(m *RuleManager) (ret *RuleManager, err error)

func WithTable(t *table.Table) Opt {
	return func(m *RuleManager) (ret *RuleManager, err error) {
		if t == nil {
			err = ErrTableMissing
			return
		}

		m.nft = t
		ret = m
		return
	}
}

func WithConfig(c *config.Config) Opt {
	return func(m *RuleManager) (ret *RuleManager, err error) {
		if c == nil {
			err = ErrConfigMissing
			return
		}

		m.cfg = c
		ret = m
		return
	}
}

func WithCgroupEventChan(ch <-chan *types.CgroupEvent) Opt {
	return func(m *RuleManager) (ret *RuleManager, err error) {
		if ch == nil {
			err = ErrCgroupEventChanMissing
			return
		}

		m.cgroupEventChan = ch
		ret = m
		return
	}
}
