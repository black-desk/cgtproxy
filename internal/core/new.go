package core

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/google/nftables"
	"github.com/sourcegraph/conc/pool"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/monitor"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/rulemanager/table"
	"github.com/black-desk/deepin-network-proxy-manager/internal/inject"
)

type Core struct {
	cfg *config.Config

	ctx       context.Context
	cancel    context.CancelFunc
	pool      pool.ErrorPool
	container *inject.Container
}

type Opt = (func(*Core) (*Core, error))

func New(opts ...Opt) (core *Core, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(
			"failed to create new deepin-network-proxy-manager core: %w",
			err,
		)
	}()

	core = &Core{}
	for i := range opts {
		core, err = opts[i](core)
		if err != nil {
			core = nil
			return
		}
	}

	if err = core.initContext(); err != nil {
		return
	}

	if err = core.initRegisterContainer(); err != nil {
		return
	}

	return
}

func (c *Core) initContext() (err error) {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	return
}

func (c *Core) initRegisterContainer() (err error) {

	c.container = inject.New()

	{
		var watcher *fsnotify.Watcher
		if watcher, err = fsnotify.NewWatcher(); err != nil {
			return
		}

		c.pool.Go(func() error {
			<-c.ctx.Done()
			return watcher.Close()
		})

		watcher.Add(c.cfg.CgroupRoot + "/...")

		if err = c.container.RegisterI(&watcher); err != nil {
			return
		}
	}

	{
		cgroupEventChan := make(chan *monitor.CgroupEvent)

		var cgroupEventChanWrite chan<- *monitor.CgroupEvent
		cgroupEventChanWrite = cgroupEventChan

		if err = c.container.Register(cgroupEventChanWrite); err != nil {
			return
		}

		var cgroupEventChanRead <-chan *monitor.CgroupEvent
		cgroupEventChanRead = cgroupEventChan

		if err = c.container.Register(cgroupEventChanRead); err != nil {
			return
		}
	}

	{
		if err = c.container.RegisterI(&c.ctx); err != nil {
			return
		}
	}

	{
		var conn *nftables.Conn

		if conn, err = nftables.New(); err != nil {
			return
		}

		var nft *table.Table
		if nft, err = table.New(
			table.WithConn(conn),
			table.WithRerouteMark(c.cfg.Mark),
			table.WithCgroupRoot(c.cfg.CgroupRoot),
		); err != nil {
			return
		}
		if err = c.container.Register(nft); err != nil {
			return
		}
	}

	{
		if err = c.container.Register(&c.cfg); err != nil {
			return
		}
	}

	{
		if err = c.container.Register(&c.pool); err != nil {
			return
		}
	}

	return
}

func WithConfig(cfg *config.Config) Opt {
	return func(core *Core) (ret *Core, err error) {
		if err = cfg.Check(); err != nil {
			return
		}

		core.cfg = cfg
		ret = core

		return
	}
}
