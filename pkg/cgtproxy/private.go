package cgtproxy

import (
	"context"
)

func (c *CGTProxy) runRouteManager(ctx context.Context) (err error) {
	defer c.log.Debugw("Route manager exited.")

	c.log.Debugw("Start route manager.")

	err = c.rtManager.RunRouteManager(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}

func (c *CGTProxy) runCGroupMonitor(ctx context.Context) (err error) {
	defer c.log.Debug("Filesystem watcher exited.")

	c.log.Debug("Start filesystem watcher.")

	err = c.cgMonitor.RunCGroupMonitor(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}
