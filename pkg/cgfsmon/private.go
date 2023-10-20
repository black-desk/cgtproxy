package cgfsmon

import (
	"context"
	"errors"
	"os"

	"github.com/black-desk/cgtproxy/pkg/types"
	fsevents "github.com/tywkeene/go-fsevents"
)

type handle struct {
	ctx context.Context
	mon *CGroupFSMonitor
}

func (h *handle) Handle(w *fsevents.Watcher, event *fsevents.FsEvent) error {
	isDirRemoved := event.IsDirRemoved()
	isDirCreated := event.IsDirCreated()
	path := event.Path

	h.mon.log.Debugw("Handling new filesystem event.",
		"event", event,
		"isDirRemoved", isDirRemoved,
		"isDirCreated", isDirCreated,
		"path", path,
	)

	func() {
		if !isDirCreated {
			return
		}

		h.mon.log.Debugw("Add path to watcher recursively.",
			"path", path,
		)

		err := w.RecursiveAdd(
			path,
			fsevents.DirCreatedEvent|fsevents.DirRemovedEvent,
		)

		h.mon.log.Debugw("Finish add path to watcher recursively.",
			"path", path,
		)

		if err == nil {
			return
		}

		if errors.Is(err, os.ErrNotExist) {
			h.mon.log.Debugw("Try to add a non-exist path to watcher.",
				"path", path,
			)
		} else {
			h.mon.log.Errorw("Failed to add path to watcher.",
				"path", path,
				"error", err,
			)
		}
	}()

	func() {
		if !isDirRemoved {
			return
		}

		err := w.RemoveDescriptor(path)
		if err == nil {
			return
		}

		h.mon.log.Errorw("Failed to remove descriptor from watcher.",
			"path", path,
			"error", err,
		)

		return
	}()

	func() {
		if !isDirCreated && !isDirRemoved {
			return
		}

		eventType := types.CgroupEventTypeNew
		if isDirRemoved {
			eventType = types.CgroupEventTypeDelete
		}

		h.mon.send(h.ctx, &types.CGroupEvent{
			Path:      path,
			EventType: eventType,
		})
	}()

	return nil
}

func (h *handle) GetMask() uint32 {
	return 1 // WHAT THE FUCK
}

func (h *handle) Check(event *fsevents.FsEvent) bool {
	return event.IsDirCreated() || event.IsDirRemoved()
}
