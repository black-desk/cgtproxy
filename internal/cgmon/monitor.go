package cgmon

import (
	"github.com/black-desk/cgtproxy/internal/fswatcher"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/errwrap"
	"go.uber.org/zap"
)

type Monitor struct {
	watcher *fswatcher.Watcher
	output  chan<- *types.CgroupEvent
	root    config.CgroupRoot
	log     *zap.SugaredLogger
}

func New(opts ...Opt) (ret *Monitor, err error) {
	defer Wrap(&err, "Failed to create the cgroup monitor.")

	m := &Monitor{}
	for i := range opts {
		m, err = opts[i](m)
		if err != nil {
			return
		}
	}

	{
		if m.watcher == nil {
			err = ErrWatcherMissing
			return
		}

		if m.output == nil {
			err = ErrOutputMissing
			return
		}

		if m.root == "" {
			err = ErrCgroupRootMissing
			return
		}

		if m.log == nil {
			m.log = zap.NewNop().Sugar()
		}
	}

	ret = m

	m.log.Debugw("Create a new cgroup monitor.")

	return
}

type Opt func(mon *Monitor) (ret *Monitor, err error)

func WithWatcher(w *fswatcher.Watcher) Opt {
	return func(mon *Monitor) (ret *Monitor, err error) {
		if w == nil {
			err = ErrWatcherMissing
			return
		}
		mon.watcher = w
		ret = mon
		return
	}
}

func WithOutput(ch chan<- *types.CgroupEvent) Opt {
	return func(mon *Monitor) (ret *Monitor, err error) {
		if ch == nil {
			err = ErrOutputMissing
			return
		}
		mon.output = ch
		ret = mon
		return
	}
}

func WithCgroupRoot(root config.CgroupRoot) Opt {
	return func(mon *Monitor) (ret *Monitor, err error) {
		if root == "" {
			err = ErrCgroupRootMissing
			return
		}
		mon.root = root
		ret = mon
		return
	}
}

func WithLogger(logger *zap.SugaredLogger) Opt {
	return func(mon *Monitor) (ret *Monitor, err error) {
		mon.log = logger
		ret = mon
		return
	}
}
