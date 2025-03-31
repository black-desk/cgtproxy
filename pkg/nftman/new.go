package nftman

import (
	"net"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/black-desk/cgtproxy/pkg/nftman/connector"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"go.uber.org/zap"
)

type NFTManager struct {
	cgroupRoot config.CGroupRoot
	bypassIPv4 []string
	bypassIPv6 []string
	log        *zap.SugaredLogger

	connector interfaces.NetlinkConnector

	table *nftables.Table

	ipv4BypassSet *nftables.Set
	ipv6BypassSet *nftables.Set

	// NOTE(black_desk):
	// When use AddSet to add anonymous protoSet into nftable,
	// we should reset protoSet.ID to 0
	// to let nftables reallocate ID for this anonymous set.
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

type Opt = (func(*NFTManager) (*NFTManager, error))

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/pkg/nftman.NFTManager -as interfaces.NFTManager -o ../interfaces/nftman.go
func New(opts ...Opt) (ret *NFTManager, err error) {
	defer Wrap(&err, "create nft table mananger")

	t := &NFTManager{}

	for i := range opts {
		t, err = opts[i](t)
		if err != nil {
			t = nil
			return
		}
	}

	if t.connector == nil {
		var ctr *connector.Connector
		ctr, err = connector.New()
		if err != nil {
			return
		}

		t.connector = ctr
	}

	if t.log == nil {
		t.log = zap.NewNop().Sugar()
	}

	ret = t
	t.log.Debugw("NFTManager created.")

	return
}

func WithBypass(bypass config.Bypass) Opt {
	return func(table *NFTManager) (ret *NFTManager, err error) {
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
				panic("this should never happened, check validator.")
			}
		}

		return table, nil
	}
}

func WithCgroupRoot(root config.CGroupRoot) Opt {
	return func(table *NFTManager) (ret *NFTManager, err error) {
		if root == "" {
			err = ErrCGroupRootMissing
			return
		}

		table.cgroupRoot = root
		return table, nil
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(nft *NFTManager) (ret *NFTManager, err error) {
		if log == nil {
			err = ErrLoggerMissing
			return
		}

		nft.log = log
		return nft, nil
	}
}

func WithConnFactory(f interfaces.NetlinkConnector) Opt {
	return func(nft *NFTManager) (ret *NFTManager, err error) {
		if f == nil {
			err = ErrConnFactoryMissing
			return
		}

		nft.connector = f
		ret = nft
		return
	}
}
