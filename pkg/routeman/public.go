// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"context"
	"errors"

	"github.com/black-desk/cgtproxy/pkg/types"
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
	err = m.initializeNftableRuels()
	if err != nil {
		return
	}

	for events := range m.cgroupEventsChan {
		newCGroups := []string{}
		deleteCGroups := []string{}

		for i := range events.Events {
			event := &events.Events[i]

			switch event.EventType {
			case types.CgroupEventTypeNew:
				newCGroups = append(newCGroups, event.Path)
			case types.CgroupEventTypeDelete:
				deleteCGroups = append(deleteCGroups, event.Path)
			}
		}

		newErr := m.handleNewCgroups(newCGroups)
		delErr := m.handleDeleteCgroups(deleteCGroups)
		eventsErr := errors.Join(newErr, delErr)

		if events.Result != nil {
			events.Result <- eventsErr
			close(events.Result)
		}
	}

	<-ctx.Done()
	return context.Cause(ctx)
}
