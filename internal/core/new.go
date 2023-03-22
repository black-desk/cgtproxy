package core

import (
	"context"
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/demo/fswatch"
	"github.com/black-desk/deepin-network-proxy-manager/internal/fswatch"
	"github.com/black-desk/deepin-network-proxy-manager/internal/inject"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/go-playground/validator/v10"
	"github.com/sourcegraph/conc/pool"
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
			"Failed to create new deepin-network-proxy-manager core: %w",
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

	if err = core.initChecks(); err != nil {
		return
	}

	if err = core.initContext(); err != nil {
		return
	}

	if err = core.initRegisterContainer(); err != nil {
		return
	}

	return
}

func (c *Core) initChecks() (err error) {
	if c.cfg == nil {
		return fmt.Errorf("Config is required.")
	}

	return
}

func (c *Core) initContext() (err error) {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	return
}

func (c *Core) initRegisterContainer() (err error) {

	var watcher fswatch.FsWatcher

	if watcher, err = demofswatch.New(
		demofswatch.WithContext(c.ctx),
		demofswatch.WithPath(c.cfg.CgroupRoot),
		demofswatch.WithRecursive(),
		demofswatch.WithEvents([]string{
			"Created",
		}),
	); err != nil {
		return
	}

	c.container = inject.New()

	err = c.container.RegisterI(&watcher)

	return
}

func WithConfig(cfg *config.Config) Opt {
	return func(core *Core) (*Core, error) {
		var validator = validator.New()
		if err := validator.Struct(cfg); err != nil {
			return nil, fmt.Errorf("Invalid config detected: %w", err)
		}

		if cfg.Rules == nil {
			log.Warning().Printf("No rules in config.")
		}

		if cfg.Repeater == nil {
			err := checkConfigRulesWithoutRepeater(cfg)
			if err != nil {
				return nil, err
			}
		}

		core.cfg = cfg

		return core, nil
	}
}

func checkConfigRulesWithoutRepeater(cfg *config.Config) (err error) {
	for i := range cfg.Rules {
		if cfg.Rules[i].Proxy != nil {
			return fmt.Errorf(
				`rules which has "proxy" field must be used with "repeater"`,
			)
		}
	}
	return
}
