package gtproxier

import (
	"github.com/black-desk/cgtproxy/pkg/gtproxier/config"
	. "github.com/black-desk/lib/go/errwrap"
	"go.uber.org/zap"
)

type Core struct {
	cfg *config.Config

	log *zap.SugaredLogger
}

type Opt = (func(*Core) (*Core, error))

func New(opts ...Opt) (ret *Core, err error) {
	defer Wrap(&err, "create new gtproxier core")

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
