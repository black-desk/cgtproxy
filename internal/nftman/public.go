// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"errors"
	"os"

	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
)

// UpdateRoutes reconciles paths with their current cgroups, adding, replacing,
// or removing map entries. Both creation and removal notifications are hints to
// check the current state; queued events may describe an older cgroup.
func (nft *NFTManager) UpdateRoutes(routes []types.Route) (err error) {
	defer Wrap(&err, "update %d routes in nftable", len(routes))
	if len(routes) == 0 {
		return nil
	}

	next, err := nft.cgroupElementsForRoutes(routes)
	if err != nil {
		return err
	}
	return nft.updateCgroupElements(next)
}

func (nft *NFTManager) Clear() (err error) {
	defer Wrap(&err, "remove nftable.")

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

	nft.dumpNFTableRules()
	return
}

func (nft *NFTManager) Release() (err error) {
	defer Wrap(&err, "release NFTManager")
	return nft.connector.Release()
}
