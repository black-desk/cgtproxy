package table

import (
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	"github.com/google/nftables"
)

type Table struct {
	conn *nftables.Conn

	rerouteMark uint32
	cgroupRoot  string

	table *nftables.Table

	ipv4BypassSet        *nftables.Set
	ipv4BypassSetElement []nftables.SetElement

	ipv6BypassSet        *nftables.Set
	ipv6BypassSetElement []nftables.SetElement

	bypassCgroupSets map[uint32]cgroupSet // level -> cgroupSet

	protoSet        *nftables.Set
	protoSetElement []nftables.SetElement

	policy nftables.ChainPolicy

	tproxyChains []*nftables.Chain
	tproxyRules  map[string][]*nftables.Rule

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
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Catch()+
			"Error occurs while initializing nft stuff:\n%w",
			err,
		)
	}()

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

	t.initStructure()

	ret = t
	return
}

func WithConn(conn *nftables.Conn) Opt {
	return func(table *Table) (*Table, error) {
		table.conn = conn
		return table, nil
	}
}

func WithRerouteMark(mark uint32) Opt {
	return func(table *Table) (*Table, error) {
		table.rerouteMark = mark
		return table, nil
	}
}

func WithCgroupRoot(root string) Opt {
	return func(table *Table) (*Table, error) {
		table.cgroupRoot = root
		return table, nil
	}
}
