package cgtproxy

import (
	"context"
)

func (c *CGTProxy) runRouteManager(ctx context.Context) (err error) {
	defer c.log.Debugw("Rule manager exited.")

	c.log.Debugw("Start nft rule manager.")

	err = c.rtManager.Run()
	if err != nil {
		return
	}

	return ctx.Err()
}

func (c *CGTProxy) runCGroupMonitor(ctx context.Context) (err error) {
	defer c.log.Debugw("Watcher exited.")

	c.log.Debugw("Start filesystem watcher.")

	err = c.cgMonitor.Run(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}
