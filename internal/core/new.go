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
	"github.com/black-desk/deepin-network-proxy-manager/internal/location"
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
		err = fmt.Errorf(location.Catch()+
			"Failed to create new deepin-network-proxy-manager core:\n%w",
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

	err = core.initContext()
	if err != nil {
		return
	}

	err = core.initRegisterContainer()
	if err != nil {
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
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return
		}

		c.pool.Go(func() error {
			<-c.ctx.Done()
			return watcher.Close()
		})

		watcher.Add(c.cfg.CgroupRoot + "/...")

		err = c.container.Register(watcher)
		if err != nil {
			return
		}
	}

	{
		cgroupEventChan := make(chan *monitor.CgroupEvent)

		var cgroupEventChanWrite chan<- *monitor.CgroupEvent
		cgroupEventChanWrite = cgroupEventChan

		err = c.container.Register(cgroupEventChanWrite)
		if err != nil {
			return
		}

		var cgroupEventChanRead <-chan *monitor.CgroupEvent
		cgroupEventChanRead = cgroupEventChan

		err = c.container.Register(cgroupEventChanRead)
		if err != nil {
			return
		}
	}

	{
		err = c.container.RegisterI(&c.ctx)
		if err != nil {
			return
		}
	}

	{
		var conn *nftables.Conn

		conn, err = nftables.New()
		if err != nil {
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
		err = c.container.Register(nft)
		if err != nil {
			return
		}
	}

	{
		err = c.container.Register(&c.cfg)
		if err != nil {
			return
		}
	}

	{
		err = c.container.Register(&c.pool)
		if err != nil {
			return
		}
	}

	return
}

func WithConfig(cfg *config.Config) Opt {
	return func(core *Core) (ret *Core, err error) {
		core.cfg = cfg
		ret = core

		return
	}
}
