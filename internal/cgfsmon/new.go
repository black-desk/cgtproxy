// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgfsmon

import (
	"path/filepath"

	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
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
