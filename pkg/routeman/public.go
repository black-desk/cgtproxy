package routeman

import (
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
)

func (m *RouteManager) Run() (err error) {
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

	for event := range m.cgroupEventChan {
		var eventErr error

		switch event.EventType {
		case types.CgroupEventTypeNew:
			eventErr = m.handleNewCgroup(event.Path)
		case types.CgroupEventTypeDelete:
			eventErr = m.handleDeleteCgroup(event.Path)
		}

		if eventErr == nil {
			continue
		}

		if event.Result != nil {
			event.Result <- eventErr
			continue
		}

		m.log.Errorw("Failed to handle cgroup event.",
			"event", event,
			"error", eventErr,
		)
	}
	return
}
