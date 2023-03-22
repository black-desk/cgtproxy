package core

import (
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/fswatch"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
)

func (c *Core) runMonitor() (err error) {
	var monitor *monitor
	if monitor, err = c.newMonitor(); err != nil {
		return
	}

	err = monitor.run()
	return
}

type monitor struct {
	Watcher fswatch.FsWatcher `inject:"true"`
}

func (c *Core) newMonitor() (m *monitor, err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(
			"Failed to create the cgroup monitor: %w",
			err,
		)
	}()

	m = &monitor{}
	if err = c.container.Fill(m); err != nil {
		return
	}

	return
}

func (m *monitor) run() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(
			"Error occurs while running the cgroup monitor: %w",
			err,
		)
	}()

	var fsEvents <-chan *fswatch.FsEvent
	if fsEvents, err = m.Watcher.Start(); err != nil {
		return
	}

	for fsEvent := range fsEvents {
		if fsEvent.Err != nil {
			return fsEvent.Err
		}
		if fsEvent.Type == fswatch.FsEventTypeCreated {
			log.Info().Printf("New cgroup found at %s", fsEvent.Path)
		}
	}
	return
}
