package core

import (
	"context"
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/fswatch"
)

type cgroupEventType uint8

const (
	cgroupEventTypeNew cgroupEventType = iota // New
)

//go:generate stringer -type=cgroupEventType -linecomment

type cgroupEvent struct {
	Path      string
	EventType cgroupEventType
}

func (c *Core) runMonitor() (err error) {
	var monitor *monitor
	if monitor, err = c.newMonitor(); err != nil {
		return
	}

	err = monitor.run()
	return
}

type monitor struct {
	Watcher   fswatch.FsWatcher   `inject:"true"`
	Ctx       context.Context     `inject:"true"`
	EventChan chan<- *cgroupEvent `inject:"true"`
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
                close(m.EventChan)

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
			err = fsEvent.Err
			return
		}

		switch fsEvent.Type {
		case fswatch.FsEventTypeCreated:
			select {
			case <-m.Ctx.Done():
				err = m.Ctx.Err()
				return
			case m.EventChan <- &cgroupEvent{
				Path:      fsEvent.Path,
				EventType: cgroupEventTypeNew,
			}:
			}
		default:
			err = fmt.Errorf(
				"Unexpected fs event type: %v",
				fsEvent.Type,
			)
			return
		}
	}
	return
}
