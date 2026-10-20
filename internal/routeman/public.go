// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"context"
	"errors"

	"github.com/black-desk/cgtproxy/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
)

func (m *RouteManager) RunRouteManager(ctx context.Context) (err error) {
	defer Wrap(&err, "running route manager")

	defer m.removeRoute()
	err = m.addRoute()
	if err != nil {
		return
	}

	defer m.removeNftableRules()
	var initial types.CGroupEvents
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case events, ok := <-m.cgroupEventsChan:
		if !ok {
			return errors.New("cgroup monitor closed before the initial scan")
		}
		initial = events
	}
	if !initial.Initial {
		return errors.New("expected the initial cgroup scan before incremental events")
	}
	paths := make([]string, 0, len(initial.Events))
	for _, event := range initial.Events {
		paths = append(paths, event.Path)
	}
	err = m.initializeNftableRules(paths)
	if initial.Result != nil {
		initial.Result <- err
		close(initial.Result)
	}
	if err != nil {
		return
	}

	for {
		var events types.CGroupEvents
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case batch, ok := <-m.cgroupEventsChan:
			if !ok {
				<-ctx.Done()
				return context.Cause(ctx)
			}
			events = batch
		}
		paths := make([]string, 0, len(events.Events))
		for _, event := range events.Events {
			switch event.EventType {
			case types.CgroupEventTypeNew, types.CgroupEventTypeDelete:
				paths = append(paths, event.Path)
			}
		}
		eventsErr := m.nft.UpdateRoutes(m.routesForPaths(paths))

		if events.Result != nil {
			events.Result <- eventsErr
			close(events.Result)
		}
		if eventsErr != nil {
			m.log.Errorw("Failed to update cgroup routes.", "error", eventsErr)
		}
	}
}
