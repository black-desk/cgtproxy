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
	"path/filepath"
	"strings"
	"syscall"

	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"go.uber.org/zap"
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

func (nft *NFTManager) initCgroupMap(conn *nftables.Conn) (err error) {
	nft.cgroupMap = &nftables.Set{
		Table:        nft.table,
		Name:         "cgroup-vmap",
		KeyType:      nftables.TypeCGroupV2,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	nft.cgroupMapElement = make(map[string]nftables.SetElement)

	err = conn.AddSet(nft.cgroupMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	return
}

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

func (t *NFTManager) removeCgroupRootFromPath(path string) string {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, string(t.cgroupRoot)) {
		path = path[len(t.cgroupRoot):]
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

func (nft *NFTManager) genSetElement(route *types.Route) (ret nftables.SetElement, err error) {
	defer Wrap(&err, "generating set element for route %v", route)

	nft.log.Debugf("Building nft map element for cgroup %q: target=%s, chain=%q.", route.Path, route.Target.Op, route.Target.Chain)

	path := route.Path
	target := route.Target

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(path)
	if err != nil {
		return
	}

	inode := fileInfo.Sys().(*syscall.Stat_t).Ino

	path = nft.removeCgroupRootFromPath(path)

	nft.log.Debugf("Resolved cgroup %q to ID %d using stat(2).", path, inode)

	setElement := nftables.SetElement{
		Key:         binaryutil.NativeEndian.PutUint64(inode),
		VerdictData: nil,
	}

	switch target.Op {
	case types.TargetDirect:
		setElement.VerdictData = &expr.Verdict{
			Kind: expr.VerdictReturn,
		}

	case types.TargetTProxy:
		setElement.VerdictData = &expr.Verdict{
			Kind:  expr.VerdictGoto,
			Chain: target.Chain,
		}
	}

	ret = setElement
	return
}

func (t *NFTManager) addCgroupRuleForLevel(
	conn *nftables.Conn, level int,
) (
	err error,
) {
	defer Wrap(&err,
		"update output chain for level %d cgroup", level)

	exprs := []expr.Any{
		&expr.Socket{ // socket load cgroupv2 => reg 1
			Key:      expr.SocketKeyCgroupv2,
			Level:    uint32(level),
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set cgroup-map dreg 0
			SourceRegister: 1,
			IsDestRegSet:   true,
			SetName:        t.cgroupMap.Name,
			SetID:          t.cgroupMap.ID,
		},
	}

	exprs = addDebugCounter(exprs)

	rule := &nftables.Rule{
		Table: t.table,
		Chain: t.outputMangleChain,
		Exprs: exprs,
	}

	conn.AddRule(rule)
	return
}
