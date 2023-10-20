package cgtproxy

import (
	"context"

	"github.com/black-desk/cgtproxy/internal/routeman"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

type CGTProxy struct {
	cfg *config.Config

	pool   *pool.ContextPool
	log    *zap.SugaredLogger
	stopCh chan error

	components *components
}

type components struct {
	m interfaces.CGroupMonitor
	r *routeman.RouteManager
}

type Opt = (func(*CGTProxy) (*CGTProxy, error))

func New(opts ...Opt) (ret *CGTProxy, err error) {
	defer Wrap(&err, "create new cgtproxy core")

	c := &CGTProxy{}
	for i := range opts {
		c, err = opts[i](c)
		if err != nil {
			c = nil
			return
		}
	}

	if c.log == nil {
		c.log = zap.NewNop().Sugar()
	}

	if c.cfg == nil {
		err = ErrConfigMissing
		return
	}

	c.pool = pool.New().
		WithContext(context.Background()).
		WithCancelOnError()

	c.stopCh = make(chan error, 1)

	ret = c

	c.log.Debugw("Create a new core.",
		"configuration", c.cfg,
	)

	return
}

func WithConfig(cfg *config.Config) Opt {
	return func(core *CGTProxy) (ret *CGTProxy, err error) {
		core.cfg = cfg
		ret = core

		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(core *CGTProxy) (ret *CGTProxy, err error) {
		core.log = log
		ret = core
		return
	}
}
