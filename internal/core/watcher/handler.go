package watcher

import (
	"errors"
	"os"

	. "github.com/black-desk/cgtproxy/internal/log"
	fsevents "github.com/tywkeene/go-fsevents"
)

type handle struct{}

func (h *handle) Handle(w *fsevents.Watcher, event *fsevents.FsEvent) error {
	isDirRemoved := event.IsDirRemoved()
	isDirCreated := event.IsDirCreated()
	path := event.Path

	Log.Debugw("Handling new filesystem event.",
		"event", event,
		"isDirRemoved", isDirRemoved,
		"isDirCreated", isDirCreated,
		"path", path,
	)

	func() {
		if !isDirCreated {
			return
		}

		Log.Debugw("Add path to watcher recursively.",
			"path", path,
		)

		err := w.RecursiveAdd(
			path,
			fsevents.DirCreatedEvent|fsevents.DirRemovedEvent,
		)

		Log.Debugw("Finish add path to watcher recursively.",
			"path", path,
		)

		if err == nil {
			return
		}

		if errors.Is(err, os.ErrNotExist) {
			Log.Debugw("Try to add a non-exist path to watcher.",
				"path", path,
			)
		} else {
			Log.Errorw("Failed to add path to watcher.",
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

		Log.Errorw("Failed to remove descriptor from watcher.",
			"path", path,
			"error", err,
		)

		return
	}()

	func() {
		w.Events <- event
	}()

	return nil
}

func (h *handle) GetMask() uint32 {
	return fsevents.DirCreatedEvent | fsevents.DirRemovedEvent
}

func (h *handle) Check(event *fsevents.FsEvent) bool {
	return event.IsDirCreated() || event.IsDirRemoved()
}
