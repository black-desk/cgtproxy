package fswatcher

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/errwrap"
	fsevents "github.com/tywkeene/go-fsevents"
	"go.uber.org/zap"
)

type Watcher struct {
	*fsevents.Watcher
	root config.CgroupRoot
	log  *zap.SugaredLogger
}

func New(opts ...Opt) (ret *Watcher, err error) {
	defer Wrap(&err, "Failed to create a filesystem watcher.")

	w := &Watcher{}

	var watcherImpl *fsevents.Watcher
	watcherImpl, err = fsevents.NewWatcher()
	if err != nil {
		return
	}

	w.Watcher = watcherImpl

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

	err = watcherImpl.RegisterEventHandler(&handle{w.log})
	if err != nil {
		return
	}

	ret = w

	w.log.Debugw("Create a new filesystem watcher.")

	return
}

type Opt func(w *Watcher) (ret *Watcher, err error)

func WithCgroupRoot(root config.CgroupRoot) Opt {
	return func(w *Watcher) (ret *Watcher, err error) {
		w.root = root
		ret = w
		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(w *Watcher) (ret *Watcher, err error) {
		w.log = log
		ret = w
		return
	}
}
