package cgtproxy

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

type CGTProxy struct {
	cfg *config.Config

	pool *pool.ContextPool
	log  *zap.SugaredLogger

	cgMonitor interfaces.CGroupMonitor
	rtManager interfaces.RouteManager
}

type Opt = (func(*CGTProxy) (*CGTProxy, error))

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/pkg/cgtproxy.CGTProxy -as interfaces.CGTProxy -o ../interfaces/cgtproxy.go

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

func WithCGroupMonitor(mon interfaces.CGroupMonitor) Opt {
	return func(core *CGTProxy) (ret *CGTProxy, err error) {
		core.cgMonitor = mon
		ret = core
		return
	}
}

func WithRouteManager(rman interfaces.RouteManager) Opt {
	return func(core *CGTProxy) (ret *CGTProxy, err error) {
		core.rtManager = rman
		ret = core
		return
	}
}
