// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgtproxy

import (
	"context"

	. "github.com/black-desk/lib/go/errwrap"
	"github.com/sourcegraph/conc/pool"
)

func (c *CGTProxy) RunCGTProxy(ctx context.Context) (err error) {
	c.log.Debug("CGTProxy starting.")
	defer c.log.Debug("CGTProxy exiting.")
	defer Wrap(&err, "running cgtproxy core")

	pool := pool.New().
		WithContext(ctx).
		WithCancelOnError()

	pool.Go(c.runCGroupMonitor)
	pool.Go(c.runRouteManager)

	return pool.Wait()
}
