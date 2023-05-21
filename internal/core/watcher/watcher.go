package watcher

import (
	"context"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	*fsnotify.Watcher
	ctx  context.Context
	root config.CgroupRoot
}

func New(opts ...Opt) (ret *Watcher, err error) {
	w := &Watcher{}

	var watcherImpl *fsnotify.Watcher
	watcherImpl, err = fsnotify.NewWatcher()
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

	{
		if w.ctx == nil {
			err = ErrContextMissing
			return
		}

		if w.root == "" {
			err = ErrCgroupRootMissing
			return
		}
	}

	ret = w
	return
}

type Opt func(w *Watcher) (ret *Watcher, err error)

func WithCgroupRoot(root config.CgroupRoot) Opt {
	return func(w *Watcher) (ret *Watcher, err error) {
		err = w.Add(string(root) + "/...")
		if err != nil {
			return
		}

		w.root = root
		ret = w
		return
	}
}

func WithContext(ctx context.Context) Opt {
	return func(w *Watcher) (ret *Watcher, err error) {
		if ctx == nil {
			err = ErrContextMissing
			return
		}

		w.ctx = ctx
		ret = w
		return
	}

}
