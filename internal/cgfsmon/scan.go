// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgfsmon

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/fsnotify/fsnotify"
	"golang.org/x/sys/unix"
)

// reconcile rescans a subtree, installs watches, and updates known to match.
// root is the filesystem path of this scan's subtree, not necessarily m.root.
// known contains paths across the whole monitored tree and is updated in place;
// entries outside this subtree are left unchanged. Notifications only select
// where to scan: even a removal notification may refer to a recreated
// directory.
func (m *CGroupFSMonitor) reconcile(
	ctx context.Context,
	watcher *fsnotify.Watcher,
	root string,
	known map[string]struct{},
) (
	[]types.CGroupEvent,
	error,
) {
	// walk installs watches and emits New for every directory it finds,
	// including known paths. Re-emitting them lets the consumer refresh
	// cgroup IDs when a directory was deleted and recreated at the same
	// path between scans.
	events, err := m.walk(ctx, watcher, root)
	if err != nil {
		return nil, err
	}
	// seen is this scan's result; known is the previous record. Use their
	// difference to find directories that disappeared from the scanned
	// subtree.
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		seen[event.Path] = struct{}{}
	}
	for path := range known {
		// Check only root and its descendants. The slash boundary keeps
		// a scan of /a from treating /abc as a descendant and deleting
		// its record.
		if path != root && !strings.HasPrefix(
			path,
			strings.TrimRight(root, "/")+"/",
		) {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		// This path was known but is now absent: remove its watch and
		// record, and notify the consumer to recheck its route.
		// The kernel may have already removed a deleted directory's
		// watch, before fsnotify has consumed the corresponding
		// notification.
		err = watcher.Remove(path)
		if err != nil && !errors.Is(
			err,
			fsnotify.ErrNonExistentWatch,
		) && !errors.Is(
			err,
			unix.EINVAL,
		) {
			return nil, fmt.Errorf(
				"unwatch cgroup %q: %w",
				path,
				err,
			)
		}
		events = append(events, types.CGroupEvent{
			Path:      path,
			EventType: types.CgroupEventTypeDelete,
		})
		delete(known, path)
	}
	// Merge the surviving/new paths after removing stale ones above.
	// known now reflects this scan for the subtree while retaining other
	// subtrees.
	for path := range seen {
		known[path] = struct{}{}
	}
	return events, nil
}

func (m *CGroupFSMonitor) walk(
	ctx context.Context,
	watcher *fsnotify.Watcher,
	root string,
) (
	[]types.CGroupEvent,
	error,
) {
	var events []types.CGroupEvent
	err := filepath.WalkDir(root, func(
		path string, d fs.DirEntry, err error,
	) error {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if errors.Is(err, fs.ErrNotExist) && path != string(m.root) {
			m.log.Debugf(
				"Cgroup %q disappeared during the scan; skipping it.",
				path,
			)
			return nil
		}
		if err != nil {
			return fmt.Errorf("scan cgroup %q: %w", path, err)
		}
		if !d.IsDir() {
			return nil
		}

		// WalkDir invokes this callback before reading the directory.
		// Add synchronously installs its watch: existing children are
		// found by the scan, and children created after that are
		// covered by events.
		if err = watcher.Add(path); err != nil {
			if errors.Is(
				err,
				fs.ErrNotExist,
			) && path != string(m.root) {
				return filepath.SkipDir
			}
			return fmt.Errorf("watch cgroup %q: %w", path, err)
		}
		events = append(events, types.CGroupEvent{
			Path:      path,
			EventType: types.CgroupEventTypeNew},
		)
		return nil
	})
	return events, err
}
