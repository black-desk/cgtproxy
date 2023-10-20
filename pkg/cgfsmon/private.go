package cgfsmon

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/pkg/types"
	fsevents "github.com/tywkeene/go-fsevents"
)

func (m *CGroupFSMonitor) walkFn(ctx context.Context) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				m.log.Debug(
					"Cgroup had been removed.",
					"path", path,
				)
				err = nil
			}
			m.log.Errorw(
				"Errors occurred while first time going through cgroupfs.",
				"path", path,
				"error", err,
			)
			err = nil
		}

		if !d.IsDir() {
			return nil
		}

		cgEvent := &types.CGroupEvent{
			Path:      path,
			EventType: types.CgroupEventTypeNew,
		}

		err = m.send(ctx, cgEvent)
		if err != nil {
			return err
		}

		return nil
	}
}

func (m *CGroupFSMonitor) walk(ctx context.Context, path string) {
	err := filepath.WalkDir(path, m.walkFn(ctx))
	if err == nil {
		return
	}

	return
}

func (m *CGroupFSMonitor) send(ctx context.Context, cgEvent *types.CGroupEvent) (err error) {
	path := strings.TrimRight(cgEvent.Path, "/")
	cgEvent.Path = path

	if cgEvent.Path == string(m.root) {
		// NOTE: Ignore cgroup root.
		return nil
	}

	m.log.Debugw("New cgroup envent.",
		"event", cgEvent,
	)

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case m.events <- *cgEvent:
		m.log.Debugw("Cgroup event sent.",
			"path", path,
		)
	}

	return
}

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
