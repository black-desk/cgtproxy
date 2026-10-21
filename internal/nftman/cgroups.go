// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"sort"
	"strings"

	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
)

func (nft *NFTManager) cgroupElementsForRoutes(routes []types.Route) (map[string]nftables.SetElement, error) {
	next := maps.Clone(nft.cgroupMapElement)
	for i := range routes {
		path := nft.removeCgroupRootFromPath(routes[i].Path)
		element, err := nft.genSetElement(&routes[i])
		if errors.Is(err, os.ErrNotExist) {
			// The cgroup may disappear between the scan/event and stat(2).
			delete(next, path)
			continue
		}
		if err != nil {
			return nil, err
		}
		next[path] = element
	}
	return next, nil
}

func (nft *NFTManager) updateCgroupElements(next map[string]nftables.SetElement) (err error) {
	var removed, added []nftables.SetElement
	for path, old := range nft.cgroupMapElement {
		if element, ok := next[path]; !ok || !sameCgroupElement(old, element) {
			removed = append(removed, old)
		}
	}
	for path, element := range next {
		if old, ok := nft.cgroupMapElement[path]; !ok || !sameCgroupElement(old, element) {
			added = append(added, element)
		}
	}
	if len(removed) == 0 && len(added) == 0 {
		return nil
	}

	conn, err := nft.connector.Connect()
	if err != nil {
		return err
	}
	defer nft.discardOnError(&err)
	if len(removed) != 0 {
		if err = queueSetElements(nft.cgroupMap, removed, conn.SetDeleteElements); err != nil {
			return err
		}
	}
	if len(added) != 0 {
		if err = queueSetElements(nft.cgroupMap, added, conn.SetAddElements); err != nil {
			return err
		}
	}
	conn.FlushChain(nft.outputMangleChain)
	if err = nft.fillOutputCgroupRules(conn, next); err != nil {
		return err
	}
	if err = conn.Flush(); err != nil {
		return err
	}
	nft.cgroupMapElement = next
	nft.log.Infow("Cgroup routes updated.", "added", len(added), "removed", len(removed))
	nft.dumpNFTableRules()
	return nil
}

func sameCgroupElement(a, b nftables.SetElement) bool {
	return bytes.Equal(a.Key, b.Key) && *a.VerdictData == *b.VerdictData
}

func (nft *NFTManager) fillOutputCgroupRules(conn *nftables.Conn, elements map[string]nftables.SetElement) error {
	if err := nft.fillOutputMangleChain(conn, nft.outputMangleChain); err != nil {
		return err
	}
	levelSet := map[int]struct{}{}
	for path := range elements {
		levelSet[strings.Count(path, "/")] = struct{}{}
	}
	levels := make([]int, 0, len(levelSet))
	for level := range levelSet {
		levels = append(levels, level)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(levels)))
	for _, level := range levels {
		if err := nft.addCgroupRuleForLevel(conn, level); err != nil {
			return err
		}
	}
	nft.addRootFallback(conn, nft.outputMangleChain)
	return nil
}
