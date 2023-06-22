package core

import (
	"context"

	"github.com/black-desk/cgtproxy/internal/cgmon"
	"github.com/black-desk/cgtproxy/internal/fswatcher"
	"github.com/black-desk/cgtproxy/internal/routeman"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

type Core struct {
	cfg *config.Config

	pool *pool.ContextPool
	log  *zap.SugaredLogger

	components *components
}

type components struct {
	w *fswatcher.Watcher
	m *cgmon.Monitor
	r *routeman.RouteManager
}

type Opt = (func(*Core) (*Core, error))

func New(opts ...Opt) (ret *Core, err error) {
	defer Wrap(&err, "Failed to create new cgtproxy core.")

	c := &Core{}
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

	ret = c

	c.log.Debugw("Create a new core.",
		"configuration", c.cfg,
	)

	return
}

func WithConfig(cfg *config.Config) Opt {
	return func(core *Core) (ret *Core, err error) {
		core.cfg = cfg
		ret = core

		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(core *Core) (ret *Core, err error) {
		core.log = log
		ret = core
		return
	}
}
