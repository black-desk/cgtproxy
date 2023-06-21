package table

import (
	. "github.com/black-desk/cgtproxy/internal/log"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"net"
)

type Table struct {
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

	markTproxyMap *nftables.Set
	markDNSMap    *nftables.Set

	outputMangleChain *nftables.Chain
	outputNATChain    *nftables.Chain
	preroutingChain   *nftables.Chain
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

	err = t.initStructure()
	if err != nil {
		return
	}

	ret = t

	Log.Debugw("Create a nft table.")
	return
}

func WithBypass(bypass config.Bypass) Opt {
	return func(table *Table) (ret *Table, err error) {
		if bypass == nil {
			ret = table
			return
		}

		for i := range bypass {
			ip := net.ParseIP(bypass[i])

			if ip == nil {
				ip, _, err = net.ParseCIDR(bypass[i])
				if err != nil {
					return
				}
			}

			if ip.To4() != nil {
				table.bypassIPv4 = append(table.bypassIPv4, bypass[i])
			} else if ip.To16() != nil {
				table.bypassIPv6 = append(table.bypassIPv6, bypass[i])
			} else {
				panic("this should never happened")
			}
		}

		return table, nil
	}
}

func WithCgroupRoot(root config.CgroupRoot) Opt {
	return func(table *Table) (*Table, error) {
		table.cgroupRoot = root
		return table, nil
	}
}
