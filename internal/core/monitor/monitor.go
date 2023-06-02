package monitor

import (
	"context"

	"github.com/black-desk/deepin-network-proxy-manager/internal/core/watcher"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
)

type Monitor struct {
	watcher *watcher.Watcher          `inject:"true"`
	ctx     context.Context           `inject:"true"`
	output  chan<- *types.CgroupEvent `inject:"true"`
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

		if m.ctx == nil {
			err = ErrContextMissing
			return
		}

		if m.output == nil {
			err = ErrOutputMissing
			return
		}
	}

	ret = m

	Log.Debugw("Create a new cgroup monitor.")

	return
}

type Opt func(mon *Monitor) (ret *Monitor, err error)

func WithWatcher(w *watcher.Watcher) Opt {
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

func WithCtx(ctx context.Context) Opt {
	return func(mon *Monitor) (ret *Monitor, err error) {
		if ctx == nil {
			err = ErrContextMissing
			return
		}
		mon.ctx = ctx
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
