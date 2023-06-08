package watcher

import (
	"errors"
	"os"

	. "github.com/black-desk/cgtproxy/internal/log"
	fsevents "github.com/tywkeene/go-fsevents"
)

type handle struct{}

func (h *handle) Handle(w *fsevents.Watcher, event *fsevents.FsEvent) error {
	Log.Debugw("Handling new filesystem event.",
		"event", event,
	)

	go func() {
		w.Events <- event
	}()

	go func() {
		if !event.IsDirRemoved() {
			return
		}

		err := w.RemoveDescriptor(event.Path)
		if err == nil {
			return
		}

		Log.Errorw("Failed to remove descriptor from watcher.",
			"path", event.Path,
			"error", err,
		)

		return
	}()

	go func() {
		if !event.IsDirCreated() {
			return
		}

		Log.Debugw("Add path to watcher recursively.",
			"path", event.Path,
		)

		err := w.RecursiveAdd(
			event.Path,
			fsevents.DirCreatedEvent|fsevents.DirRemovedEvent,
		)
		if err == nil {
			return
		}

		if errors.Is(err, os.ErrNotExist) {
			Log.Debugw("Try to add a non-exist path to watcher.",
				"path", event.Path,
			)
		} else {
			Log.Errorw("Failed to add path to watcher.",
				"path", event.Path,
				"error", err,
			)
		}
	}()

	return nil
}

func (h *handle) GetMask() uint32 {
	return fsevents.DirCreatedEvent | fsevents.DirRemovedEvent
}

func (h *handle) Check(event *fsevents.FsEvent) bool {
	return event.IsDirCreated() || event.IsDirRemoved()
}
