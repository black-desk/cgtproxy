// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"sort"
	"strings"

	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"go.uber.org/zap"
)

func (nft *NFTManager) cgroupElementsForRoutes(routes []types.Route) (map[string]nftables.SetElement, error) {
	next := maps.Clone(nft.cgroupMapElement)
	for i := range routes {
		path := nft.removeCgroupRootFromPath(routes[i].Path)
		element, err := nft.genSetElement(&routes[i])
		if errors.Is(err, os.ErrNotExist) {
			// The cgroup may disappear between the scan/event and stat(2).
			nft.log.Debugf("Cgroup %q disappeared before its route could be built; removing any existing route.", routes[i].Path)
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
	var removedPaths, addedPaths []string
	for path, old := range nft.cgroupMapElement {
		if element, ok := next[path]; !ok || !sameCgroupElement(old, element) {
			removed = append(removed, old)
			removedPaths = append(removedPaths, path)
		}
	}
	for path, element := range next {
		if old, ok := nft.cgroupMapElement[path]; !ok || !sameCgroupElement(old, element) {
			added = append(added, element)
			addedPaths = append(addedPaths, path)
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
	nft.logCgroupRoutes("Removed", nft.cgroupMapElement, removedPaths)
	nft.logCgroupRoutes("Added", next, addedPaths)
	nft.cgroupMapElement = next
	nft.dumpNFTableRules()
	return nil
}

// Paths come from the existing route diff; logging must not scan the full map
// again. Call only after the kernel accepts the transaction.
func (nft *NFTManager) logCgroupRoutes(action string, elements map[string]nftables.SetElement, paths []string) {
	if !nft.log.Desugar().Core().Enabled(zap.InfoLevel) {
		return
	}
	for _, path := range paths {
		nft.logCgroupChange(action, path, elements[path])
	}
}

func (nft *NFTManager) logCgroupChange(action, path string, element nftables.SetElement) {
	target := "DIRECT"
	if element.VerdictData.Chain != "" {
		target = fmt.Sprintf("TPROXY chain %q", element.VerdictData.Chain)
	}
	nft.log.Infof("%s cgroup route: path=%q, id=%d, target=%s.", action, path,
		binaryutil.NativeEndian.Uint64(element.Key), target)
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
