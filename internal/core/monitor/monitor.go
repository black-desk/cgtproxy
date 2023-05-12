package monitor

import (
	"context"
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/inject"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	"github.com/fsnotify/fsnotify"
)

type CgroupEventType uint8

const (
	CgroupEventTypeNew    CgroupEventType = iota // New
	CgroupEventTypeDelete                        // Delete
)

//go:generate stringer -type=CgroupEventType -linecomment

type CgroupEvent struct {
	Path      string
	EventType CgroupEventType
}

type Monitor struct {
	Watcher   *fsnotify.Watcher   `inject:"true"`
	Ctx       context.Context     `inject:"true"`
	EventChan chan<- *CgroupEvent `inject:"true"`
}

func New(container *inject.Container) (m *Monitor, err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Capture()+
			"Failed to create the cgroup monitor:\n%w",
			err,
		)
	}()

	m = &Monitor{}
	err = container.Fill(m)
	if err != nil {
		return
	}

	return
}

func (m *Monitor) Run() (err error) {
	defer func() {
		close(m.EventChan)

		if err == nil {
			return
		}

		err = fmt.Errorf(location.Capture()+
			"Error occurs while running the cgroup monitor:\n%w",
			err,
		)
	}()

	// TODO(black_desk): handle exsiting cgroup

	var cgEvent *CgroupEvent
	for fsEvent := range m.Watcher.Events {
		if fsEvent.Has(fsnotify.Create) {
			cgEvent = &CgroupEvent{
				Path:      fsEvent.Name,
				EventType: CgroupEventTypeNew,
			}
		} else if fsEvent.Has(fsnotify.Remove) {
			cgEvent = &CgroupEvent{
				Path:      fsEvent.Name,
				EventType: CgroupEventTypeDelete,
			}

		} else if fsEvent.Has(fsnotify.Chmod) {
			// We not care about this kind of event
		} else if fsEvent.Has(fsnotify.Write) {
			// We not care about this kind of event
		} else if fsEvent.Has(fsnotify.Rename) {
			// We not care about this kind of event
		} else {
			err = fmt.Errorf(location.Capture()+
				"%w: %v", ErrUnexpectFsEventType,
				fsEvent.Op.String(),
			)
			return
		}

		select {
		case <-m.Ctx.Done():
			err = m.Ctx.Err()
			return
		case m.EventChan <- cgEvent:
		}
	}
	return
}
