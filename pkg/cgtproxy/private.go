package cgtproxy

import (
	"context"
)

func (c *CGTProxy) runRuleManager(ctx context.Context) (err error) {
	defer c.log.Debugw("Rule manager exited.")

	c.log.Debugw("Start nft rule manager.")

	err = c.components.r.Run()
	if err != nil {
		return
	}

	return ctx.Err()
}

func (c *CGTProxy) runWatcher(ctx context.Context) (err error) {
	defer c.log.Debugw("Watcher exited.")

	c.log.Debugw("Start filesystem watcher.")

	err = c.components.m.Run(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}
