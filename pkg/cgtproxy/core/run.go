package core

import (
	"context"

	. "github.com/black-desk/lib/go/errwrap"
)

func (c *Core) Run() (err error) {
	defer Wrap(&err, "running cgtproxy core")

	c.components, err = injectedComponents(c.cfg, c.log)
	if err != nil {
		return
	}

	c.pool.Go(c.waitStop)
	c.pool.Go(c.runWatcher)
	c.pool.Go(c.runMonitor)
	c.pool.Go(c.runRuleManager)

	return c.pool.Wait()
}

func (c *Core) Stop(err error) {
	c.stopCh <- err
}

func (c *Core) waitStop(ctx context.Context) (err error) {
	defer Wrap(&err)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-c.stopCh:
		c.log.Debug("Stopped.")
		return err
	}
}

func (c *Core) runMonitor(ctx context.Context) (err error) {
	defer c.log.Debugw("Cgroup monitor exited.")

	c.log.Debugw("Start cgroup monitor.")

	err = c.components.m.Run(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}

func (c *Core) runRuleManager(ctx context.Context) (err error) {
	defer c.log.Debugw("Rule manager exited.")

	c.log.Debugw("Start nft rule manager.")

	err = c.components.r.Run()
	if err != nil {
		return
	}

	return ctx.Err()
}

func (c *Core) runWatcher(ctx context.Context) (err error) {
	defer c.log.Debugw("Watcher exited.")

	c.log.Debugw("Start filesystem watcher.")

	err = c.components.w.Run(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}
