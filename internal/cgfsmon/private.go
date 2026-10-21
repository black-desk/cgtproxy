// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
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

func (m *CGroupFSMonitor) walk(ctx context.Context, watcher *fsnotify.Watcher, root string) ([]types.CGroupEvent, error) {
	var events []types.CGroupEvent
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if errors.Is(err, fs.ErrNotExist) && path != string(m.root) {
			m.log.Debugw("Cgroup had been removed.", "path", path)
			return nil
		}
		if err != nil {
			return fmt.Errorf("scan cgroup %q: %w", path, err)
		}
		if !d.IsDir() {
			return nil
		}

		// WalkDir invokes this callback before reading the directory. Add
		// synchronously installs its watch: existing children are found by
		// the scan, and children created after that are covered by events.
		if err = watcher.Add(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) && path != string(m.root) {
				return filepath.SkipDir
			}
			return fmt.Errorf("watch cgroup %q: %w", path, err)
		}
		if path != string(m.root) {
			events = append(events, types.CGroupEvent{Path: path, EventType: types.CgroupEventTypeNew})
		}
		return nil
	})
	return events, err
}

func (m *CGroupFSMonitor) reconcile(ctx context.Context, watcher *fsnotify.Watcher, root string, known map[string]struct{}) ([]types.CGroupEvent, error) {
	events, err := m.walk(ctx, watcher, root)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		seen[event.Path] = struct{}{}
	}
	for path := range known {
		if path != root && !strings.HasPrefix(path, strings.TrimRight(root, "/")+"/") {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		// The kernel may have already removed a deleted directory's watch,
		// before fsnotify has consumed the corresponding notification.
		if err = watcher.Remove(path); err != nil && !errors.Is(err, fsnotify.ErrNonExistentWatch) && !errors.Is(err, unix.EINVAL) {
			return nil, fmt.Errorf("unwatch cgroup %q: %w", path, err)
		}
		events = append(events, types.CGroupEvent{Path: path, EventType: types.CgroupEventTypeDelete})
		delete(known, path)
	}
	for path := range seen {
		known[path] = struct{}{}
	}
	return events, nil
}

func (m *CGroupFSMonitor) send(ctx context.Context, cgEvents types.CGroupEvents) (err error) {
	for i := range cgEvents.Events {
		path := strings.TrimRight(cgEvents.Events[i].Path, "/")
		cgEvents.Events[i].Path = path
	}

	m.log.Debugw("New cgroup events.", "size", len(cgEvents.Events))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.eventsOut <- cgEvents:
		m.log.Debugw("Cgroup events sent.", "size", len(cgEvents.Events))
	}
	return nil
}
