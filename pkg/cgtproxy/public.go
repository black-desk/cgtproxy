package cgtproxy

import (
	"context"

	. "github.com/black-desk/lib/go/errwrap"
	"github.com/sourcegraph/conc/pool"
)

func (c *CGTProxy) Run(ctx context.Context) (err error) {
	defer Wrap(&err, "running cgtproxy core")

	c.components, err = injectedComponents(c.cfg, c.log)
	if err != nil {
		return
	}

	pool := pool.New().
		WithContext(ctx).
		WithCancelOnError()

	pool.Go(c.runWatcher)
	pool.Go(c.runRuleManager)

	return pool.Wait()
}
