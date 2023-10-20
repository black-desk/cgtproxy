package cgfsmon

import (
	"errors"
	"os"

	"github.com/black-desk/cgtproxy/pkg/types"
	fsevents "github.com/tywkeene/go-fsevents"
	"go.uber.org/zap"
)

type handle struct {
	log    *zap.SugaredLogger
	events chan<- types.CGroupEvent
}

func (h *handle) Handle(w *fsevents.Watcher, event *fsevents.FsEvent) error {
	isDirRemoved := event.IsDirRemoved()
	isDirCreated := event.IsDirCreated()
	path := event.Path

	h.log.Debugw("Handling new filesystem event.",
		"event", event,
		"isDirRemoved", isDirRemoved,
		"isDirCreated", isDirCreated,
		"path", path,
	)

	func() {
		if !isDirCreated {
			return
		}

		h.log.Debugw("Add path to watcher recursively.",
			"path", path,
		)

		err := w.RecursiveAdd(
			path,
			fsevents.DirCreatedEvent|fsevents.DirRemovedEvent,
		)

		h.log.Debugw("Finish add path to watcher recursively.",
			"path", path,
		)

		if err == nil {
			return
		}

		if errors.Is(err, os.ErrNotExist) {
			h.log.Debugw("Try to add a non-exist path to watcher.",
				"path", path,
			)
		} else {
			h.log.Errorw("Failed to add path to watcher.",
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

		h.log.Errorw("Failed to remove descriptor from watcher.",
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

		h.events <- types.CGroupEvent{
			Path:      path,
			EventType: eventType,
		}
	}()

	return nil
}

func (h *handle) GetMask() uint32 {
	return 1 // WHAT THE FUCK
}

func (h *handle) Check(event *fsevents.FsEvent) bool {
	return event.IsDirCreated() || event.IsDirRemoved()
}
