package cgfsmon

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/rjeczalik/notify"
	"go.uber.org/zap"
)

type CGroupFSMonitor struct {
	eventsOut chan types.CGroupEvent
	eventsIn  chan notify.EventInfo
	root      config.CGroupRoot
	log       *zap.SugaredLogger
}

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/pkg/cgfsmon.CGroupFSMonitor -as interfaces.CGroupMonitor -o ../interfaces/cgmon.go

func New(opts ...Opt) (ret *CGroupFSMonitor, err error) {
	defer Wrap(&err, "create filesystem watcher")

	w := &CGroupFSMonitor{}

	w.eventsOut = make(chan types.CGroupEvent)
	w.eventsIn = make(chan notify.EventInfo)

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

	ret = w

	w.log.Debugw("Create a new filesystem watcher.")

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
