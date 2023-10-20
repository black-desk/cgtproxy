package cgfsmon

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	fsevents "github.com/tywkeene/go-fsevents"
	"go.uber.org/zap"
)

type CGroupFSMonitor struct {
	events  chan types.CgroupEvent
	watcher *fsevents.Watcher
	root    config.CgroupRoot
	log     *zap.SugaredLogger
}

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@latest -for github.com/black-desk/cgtproxy/pkg/cgfsmon.CGroupFSMonitor -as interfaces.CGroupMonitor -o ../interfaces/cgmon.go

func New(opts ...Opt) (ret *CGroupFSMonitor, err error) {
	defer Wrap(&err, "create filesystem watcher")

	w := &CGroupFSMonitor{}

	var watcherImpl *fsevents.Watcher
	watcherImpl, err = fsevents.NewWatcher()
	if err != nil {
		return
	}

	w.events = make(chan types.CgroupEvent)

	w.watcher = watcherImpl

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
		err = ErrCgroupRootMissing
		return
	}

	err = watcherImpl.RegisterEventHandler(&handle{
		log:    w.log,
		events: w.events,
	})
	if err != nil {
		return
	}

	ret = w

	w.log.Debugw("Create a new filesystem watcher.")

	return
}

type Opt func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error)

func WithCgroupRoot(root config.CgroupRoot) Opt {
	return func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error) {
		w.root = root
		ret = w
		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error) {
		w.log = log
		ret = w
		return
	}
}
