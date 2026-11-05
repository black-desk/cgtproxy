// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"context"
	"errors"

	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/black-desk/cgtproxy/internal/types"
	"golang.org/x/exp/maps"
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
			return eventsErr
		}
	}
}

func (m *RouteManager) initializeNftableRules(paths []string) (err error) {
	defer Wrap(&err, "initializing nftable rules")

	for _, tp := range m.cfg.TProxies {
		err = m.addRule(tp)
		if err != nil {
			return
		}
	}

	return m.nft.Initialize(maps.Values(m.cfg.TProxies), m.targetForPath("/"), m.routesForPaths(paths))
}

func (m *RouteManager) removeNftableRules() {
	var err error

	err = m.nft.Clear()
	if err != nil {
		m.log.Errorf("Failed to delete the cgtproxy nft table: %v", err)
		err = nil
	}

	m.removeRules()

	err = m.nft.Release()
	if err != nil {
		m.log.Errorf("Failed to release the nftables connection: %v", err)
		err = nil
	}
}
