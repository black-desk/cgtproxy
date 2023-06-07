package table

import (
	"github.com/black-desk/cgtproxy/internal/config"
	. "github.com/black-desk/cgtproxy/internal/log"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
)

type Table struct {
	conn       *nftables.Conn
	cgroupRoot config.CgroupRoot
	bypassIPv4 []string
	bypassIPv6 []string

	table *nftables.Table

	ipv4BypassSet *nftables.Set
	ipv6BypassSet *nftables.Set

	protoSet        *nftables.Set
	protoSetElement []nftables.SetElement // keep anonymous set elements

	policy nftables.ChainPolicy

	cgroupMap        *nftables.Set
	cgroupMapElement map[string]nftables.SetElement

	markMap *nftables.Set

	outputChain     *nftables.Chain
	preroutingChain *nftables.Chain
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
		if bypass == nil {
			return table, nil
		}

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

func WithCgroupRoot(root config.CgroupRoot) Opt {
	return func(table *Table) (*Table, error) {
		table.cgroupRoot = root
		return table, nil
	}
}
