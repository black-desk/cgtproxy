// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgtproxy

import (
	"context"

	"github.com/black-desk/cgtproxy/internal/cgfsmon"
	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/routeman"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

type CGTProxy struct {
	cfg *config.Config

	pool *pool.ContextPool
	log  *zap.SugaredLogger

	cgMonitor *cgfsmon.CGroupFSMonitor
	rtManager *routeman.RouteManager
}

type Opt = (func(*CGTProxy) (*CGTProxy, error))

func New(opts ...Opt) (ret *CGTProxy, err error) {
	defer errwrap.Wrap(&err, "create new cgtproxy core")

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

	if c.cgMonitor == nil {
		err = ErrCGroupMonitorMissing
		return
	}

	if c.rtManager == nil {
		err = ErrRouteManagerMissing
		return
	}

	ret = c

	c.log.Debugf("Created cgtproxy core: cgroup root=%q, route table=%d, gateway=%+v, proxies=%d, rules=%d.",
		c.cfg.CgroupRoot, c.cfg.RouteTable, c.cfg.Gateway, len(c.cfg.TProxies), len(c.cfg.Rules))

	return
}

func WithConfig(cfg *config.Config) Opt {
	return func(core *CGTProxy) (ret *CGTProxy, err error) {
		if cfg == nil {
			err = ErrConfigMissing
			return
		}

		core.cfg = cfg
		ret = core
		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(core *CGTProxy) (ret *CGTProxy, err error) {
		if log == nil {
			err = ErrLoggerMissing
			return
		}

		core.log = log
		ret = core
		return
	}
}

func WithCGroupMonitor(mon *cgfsmon.CGroupFSMonitor) Opt {
	return func(core *CGTProxy) (ret *CGTProxy, err error) {
		if mon == nil {
			err = ErrCGroupMonitorMissing
			return
		}

		core.cgMonitor = mon
		ret = core
		return
	}
}

func WithRouteManager(rman *routeman.RouteManager) Opt {
	return func(core *CGTProxy) (ret *CGTProxy, err error) {
		if rman == nil {
			err = ErrRouteManagerMissing
			return
		}

		core.rtManager = rman
		ret = core
		return
	}
}

func (c *CGTProxy) Run(ctx context.Context) (err error) {
	c.log.Debug("CGTProxy starting.")
	defer c.log.Debug("CGTProxy exiting.")

	pool := pool.New().
		WithContext(ctx).
		WithCancelOnError()

	pool.Go(c.runCGroupMonitor)
	pool.Go(c.runRouteManager)

	return pool.Wait()
}

func (c *CGTProxy) runRouteManager(ctx context.Context) (err error) {
	c.log.Debug("Start route manager.")
	defer c.log.Debug("Route manager exited.")

	err = c.rtManager.Run(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}

func (c *CGTProxy) runCGroupMonitor(ctx context.Context) (err error) {
	c.log.Debug("Start filesystem watcher.")
	defer c.log.Debug("Filesystem watcher exited.")

	err = c.cgMonitor.Run(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}
