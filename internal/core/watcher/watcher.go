package watcher

import (
	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	. "github.com/black-desk/lib/go/errwrap"
	fsevents "github.com/tywkeene/go-fsevents"
)

type Watcher struct {
	*fsevents.Watcher
	root config.CgroupRoot
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

	if w.root == "" {
		err = ErrCgroupRootMissing
		return
	}

	ret = w

	Log.Debugw("Create a new filesystem watcher.")

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
