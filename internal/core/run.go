package core

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	. "github.com/black-desk/cgtproxy/internal/log"
	. "github.com/black-desk/lib/go/errwrap"
)

func (c *Core) Run() (err error) {
	defer Wrap(&err, "Error occurs while running the core.")

	c.components, err = injectedComponents(c.cfg)
	if err != nil {
		return
	}

	c.pool.Go(c.waitSig)
	c.pool.Go(c.runWatcher)
	c.pool.Go(c.runMonitor)
	c.pool.Go(c.runRuleManager)

	return c.pool.Wait()
}

func (c *Core) waitSig(ctx context.Context) (err error) {
	defer Wrap(&err)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	var sig os.Signal
	select {
	case <-ctx.Done():
		return ctx.Err()
	case sig = <-sigChan:
		Log.Debugw(
			"Receive signal.",
			"signal", sig,
		)
		return &ErrCancelBySignal{sig}
	}
}

func (c *Core) runMonitor(ctx context.Context) (err error) {
	defer Log.Debugw("Cgroup monitor exited.")

	Log.Debugw("Start cgroup monitor.")

	err = c.components.m.Run(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}

func (c *Core) runRuleManager(ctx context.Context) (err error) {
	defer Log.Debugw("Rule manager exited.")

	Log.Debugw("Start nft rule manager.")

	err = c.components.r.Run()
	if err != nil {
		return
	}

	return ctx.Err()
}

func (c *Core) runWatcher(ctx context.Context) (err error) {
	defer Log.Debugw("Watcher exited.")

	Log.Debugw("Start filesystem watcher.")

	err = c.components.w.Run(ctx)
	if err != nil {
		return
	}

	return ctx.Err()
}
