package core

import (
	"context"
	"fmt"

	"github.com/sourcegraph/conc/pool"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
)

type Core struct {
	cfg *config.Config

	pool   pool.ErrorPool
	ctx    context.Context
	cancel context.CancelFunc
}

type Opt = (func(*Core) (*Core, error))

func New(opts ...Opt) (ret *Core, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Capture()+
			"Failed to create new deepin-network-proxy-manager core:\n%w",
			err,
		)
	}()

	core := &Core{}
	for i := range opts {
		core, err = opts[i](core)
		if err != nil {
			core = nil
			return
		}
	}

	if core.cfg == nil {
		err = ErrConfigMissing
		return
	}

	err = core.initContext()
	if err != nil {
		return
	}

	ret = core
	return
}

func (c *Core) initContext() (err error) {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	return
}

func WithConfig(cfg *config.Config) Opt {
	return func(core *Core) (ret *Core, err error) {
		if cfg == nil {
			err = ErrConfigMissing
			return
		}

		core.cfg = cfg
		ret = core

		return
	}
}
