package core

import (
	"context"

	"github.com/sourcegraph/conc/pool"

	"github.com/black-desk/cgtproxy/internal/config"
	. "github.com/black-desk/cgtproxy/internal/log"
	. "github.com/black-desk/lib/go/errwrap"
)

type Core struct {
	cfg *config.Config

	pool *pool.ContextPool
}

type Opt = (func(*Core) (*Core, error))

func New(opts ...Opt) (ret *Core, err error) {
	defer Wrap(&err, "Failed to create new cgtproxy core.")

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

	core.pool = pool.New().
		WithContext(context.Background()).
		WithCancelOnError()

	ret = core

	Log.Debugw("Create a new core.",
		"configuration", core.cfg,
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
