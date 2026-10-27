// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgfsmon

import (
	"context"
	"errors"

	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/fsnotify/fsnotify"
)

func (w *CGroupFSMonitor) Events() <-chan types.CGroupEvents {
	return w.eventsOut
}

func (w *CGroupFSMonitor) RunCGroupMonitor(ctx context.Context) (err error) {
	defer Wrap(&err, "running filesystem watcher")
	defer close(w.eventsOut)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() {
		if watcher != nil {
			_ = watcher.Close()
		}
	}()

	known := make(map[string]struct{})
	w.log.Infof("Scanning cgroup filesystem at %q and installing watches.", w.root)
	events, err := w.reconcile(ctx, watcher, string(w.root), known)
	if err != nil {
		return err
	}
	w.log.Infof("Initial cgroup scan completed: found %d cgroups under %q.", len(events), w.root)
	if err = w.send(ctx, types.CGroupEvents{Initial: true, Events: events}); err != nil {
		return err
	}

	for {
		var path string
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return errors.New("filesystem watcher closed unexpectedly")
			}
			if !errors.Is(watchErr, fsnotify.ErrEventOverflow) {
				return watchErr
			}
			w.log.Warn("Filesystem event queue overflowed; rebuilding watches and rescanning cgroupfs.")
			// Overflow may also lose watch-removal events. Rebuild the watcher
			// rather than trusting its bookkeeping for deleted/recreated paths.
			if err = watcher.Close(); err != nil {
				return err
			}
			watcher, err = fsnotify.NewWatcher()
			if err != nil {
				return err
			}
			path = string(w.root)
		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("filesystem watcher closed unexpectedly")
			}
			if !event.Has(fsnotify.Create | fsnotify.Remove | fsnotify.Rename) {
				continue
			}
			w.log.Debugf("Received filesystem event %s for %q; rescanning the subtree.", event.Op, event.Name)
			path = event.Name
		}

		// Events can be stale: scan the current subtree even on removal, so
		// deleting and recreating a path cannot leave its children unwatched.
		events, err = w.reconcile(ctx, watcher, path, known)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			continue
		}
		if err = w.send(ctx, types.CGroupEvents{Events: events}); err != nil {
			return err
		}
	}
}
