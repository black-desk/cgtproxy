package cgtproxy

import (
	"context"

	. "github.com/black-desk/lib/go/errwrap"
	"github.com/sourcegraph/conc/pool"
)

func (c *CGTProxy) Run(ctx context.Context) (err error) {
	defer Wrap(&err, "running cgtproxy core")

	pool := pool.New().
		WithContext(ctx).
		WithCancelOnError()

	pool.Go(c.runCGroupMonitor)
	pool.Go(c.runRouteManager)

	return pool.Wait()
}
