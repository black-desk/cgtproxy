// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgfsmon

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type CGroupFSMonitor struct {
	eventsOut chan types.CGroupEvents
	root      config.CGroupRoot
	log       *zap.SugaredLogger
}

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/internal/cgfsmon.CGroupFSMonitor -as interfaces.CGroupMonitor -o ../interfaces/cgmon.go

func New(opts ...Opt) (ret *CGroupFSMonitor, err error) {
	defer Wrap(&err, "create filesystem watcher")

	w := &CGroupFSMonitor{}

	w.eventsOut = make(chan types.CGroupEvents)

	for i := range opts {
		w, err = opts[i](w)
		if err != nil {
			return
		}
	}

	if w.log == nil {
		w.log = zap.NewNop().Sugar()
	}

	if w.root == "" {
		err = ErrCGroupRootNotFound
		return
	}
	w.root = config.CGroupRoot(filepath.Clean(string(w.root)))

	ret = w

	w.log.Debug("Create a cgroupv2 filesystem monitor.")

	return
}

type Opt func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error)

func WithCgroupRoot(root config.CGroupRoot) Opt {
	return func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error) {
		if root == "" {
			err = ErrCGroupRootNotFound
			return
		}

		w.root = root
		ret = w
		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error) {
		if log == nil {
			err = ErrLoggerMissing
			return
		}

		w.log = log
		ret = w
		return
	}
}

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

func (m *CGroupFSMonitor) send(ctx context.Context, cgEvents types.CGroupEvents) (err error) {
	for i := range cgEvents.Events {
		path := strings.TrimRight(cgEvents.Events[i].Path, "/")
		cgEvents.Events[i].Path = path
	}

	m.log.Debugf("Sending %d cgroup events (initial scan: %t) to the route manager.", len(cgEvents.Events), cgEvents.Initial)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.eventsOut <- cgEvents:
		m.log.Debugf("Sent %d cgroup events to the route manager.", len(cgEvents.Events))
	}
	return nil
}
