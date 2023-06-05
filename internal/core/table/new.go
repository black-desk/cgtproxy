package table

import (
	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
)

type Table struct {
	conn        *nftables.Conn
	rerouteMark config.RerouteMark
	cgroupRoot  config.CgroupRoot
	bypassIPv4  []string
	bypassIPv6  []string

	table *nftables.Table

	ipv4BypassSet *nftables.Set
	ipv6BypassSet *nftables.Set

	bypassCgroupSets map[uint32]cgroupSet // level -> cgroupSet

	protoSet        *nftables.Set
	protoSetElement []nftables.SetElement // keep anonymous set elements

	policy nftables.ChainPolicy

	tproxyChains []*nftables.Chain

	cgroupMaps map[uint32]cgroupSet // level -> cgroupSet

	outputChain     *nftables.Chain
	outputRules     []*nftables.Rule
	preroutingChain *nftables.Chain
	preroutingRules []*nftables.Rule
}

type cgroupSet struct {
	set      *nftables.Set
	elements map[string]nftables.SetElement
}

type Opt = (func(*Table) (*Table, error))

func New(opts ...Opt) (ret *Table, err error) {
	defer Wrap(&err, "Failed to create nft table.")

	t := &Table{}

	for i := range opts {
		t, err = opts[i](t)
		if err != nil {
			t = nil
			return
		}
	}

	err = t.initChecks()
	if err != nil {
		return
	}

	err = t.initStructure()
	if err != nil {
		return
	}

	ret = t

	Log.Debugw("Create a nft table.")
	return
}

func WithBypass(bypass *config.Bypass) Opt {
	return func(table *Table) (*Table, error) {
		table.bypassIPv4 = bypass.IPV4
		table.bypassIPv6 = bypass.IPV6
		return table, nil
	}
}

func WithConn(conn *nftables.Conn) Opt {
	return func(table *Table) (*Table, error) {
		table.conn = conn
		return table, nil
	}
}

func WithRerouteMark(mark config.RerouteMark) Opt {
	return func(table *Table) (*Table, error) {
		table.rerouteMark = mark
		return table, nil
	}
}

func WithCgroupRoot(root config.CgroupRoot) Opt {
	return func(table *Table) (*Table, error) {
		table.cgroupRoot = root
		return table, nil
	}
}
