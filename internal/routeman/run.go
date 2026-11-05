// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"context"

	"github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
)

func (m *RouteManager) Run(ctx context.Context) (err error) {
	defer errwrap.Wrap(&err, "running route manager")

	defer m.removeRoute()
	err = m.addRoute()
	if err != nil {
		return
	}

	defer m.removeRules()
	for _, tp := range m.cfg.TProxies {
		if err = m.addRule(tp); err != nil {
			return err
		}
	}

	defer m.removeNftableRules()

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
			return eventsErr
		}
	}
}

func (m *RouteManager) removeNftableRules() {
	var err error

	err = m.nft.Clear()
	if err != nil {
		m.log.Errorf("Failed to delete the cgtproxy nft table: %v", err)
		err = nil
	}

	err = m.nft.Release()
	if err == nil {
		return
	}

	m.log.Errorf("Failed to release the nftables connection: %v", err)
}
