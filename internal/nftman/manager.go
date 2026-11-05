// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"errors"
	"net"
	"os"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/nftman/connector"
	"github.com/google/nftables"
	"go.uber.org/zap"
)

type NFTManager struct {
	cgroupRoot config.CGroupRoot
	bypassIPv4 []string
	bypassIPv6 []string
	gateway    bool
	log        *zap.SugaredLogger

	connector *connector.Connector

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
	proxyMarkSet  *nftables.Set

	outputMangleChain *nftables.Chain
	outputNATChain    *nftables.Chain
	preroutingChain   *nftables.Chain
}

type Opt = (func(*NFTManager) (*NFTManager, error))

func New(opts ...Opt) (ret *NFTManager, err error) {
	defer errwrap.Wrap(&err, "create nftables manager")

	t := &NFTManager{connector: &connector.Connector{}}

	for i := range opts {
		t, err = opts[i](t)
		if err != nil {
			t = nil
			return
		}
	}

	if t.log == nil {
		t.log = zap.NewNop().Sugar()
	}

	ret = t
	t.log.Debug("NFTManager created.")

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

func WithGateway(enabled bool) Opt {
	return func(nft *NFTManager) (*NFTManager, error) {
		nft.gateway = enabled
		return nft, nil
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

func (nft *NFTManager) Clear() (err error) {
	defer errwrap.Wrap(&err, "remove nftables table")

	if nft.table == nil {
		return
	}

	var conn *nftables.Conn
	conn, err = nft.connector.Connect()
	if err != nil {
		return
	}

	conn.DelTable(nft.table)
	err = conn.Flush()
	if errors.Is(err, os.ErrNotExist) {
		nft.log.Debugf("Nft table %q is already absent; skipping removal.", nft.table.Name)
		err = nil
	} else if err != nil {
		return
	}

	return
}

func (nft *NFTManager) Release() (err error) {
	return nft.connector.Release()
}
