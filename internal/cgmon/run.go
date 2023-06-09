package cgmon

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
)

func (m *Monitor) walkFn(ctx context.Context) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		cgEvent := &types.CgroupEvent{
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

func (m *Monitor) walk(ctx context.Context, path string) {
	err := m.doWalk(ctx, path)
	if err == nil {
		return
	}

	if errors.Is(err, fs.ErrNotExist) {
		m.log.Debug("Cgroup had been removed.",
			"path", path,
		)
		return
	}

	m.log.Errorw("Errors occurred.",
		"path", path,
		"error", err,
	)
	return
}

func (m *Monitor) doWalk(ctx context.Context, path string) (err error) {
	err = filepath.WalkDir(path, m.walkFn(ctx))
	return
}

func (m *Monitor) Run(ctx context.Context) (err error) {
	defer close(m.output)
	defer Wrap(&err, "running cgroup monitor.")

	m.log.Debugw("Initializing cgroup monitor...")

	err = m.doWalk(ctx, string(m.root))
	if err != nil {
		return
	}

	m.log.Debugw("Initializing cgroup monitor done.")

	var cgEvent *types.CgroupEvent
	for {
		select {
		case fsEvent, ok := <-m.watcher.Events:
			if !ok {
				m.log.Debugw(
					"Filesystem watcher channel closed.",
				)
				return
			}

			m.log.Debugw("New filesystem envent arrived.",
				"event", fsEvent,
			)

			if fsEvent.IsDirCreated() {
				cgEvent = &types.CgroupEvent{
					Path:      fsEvent.Path,
					EventType: types.CgroupEventTypeNew,
				}

				m.walk(ctx, fsEvent.Path)

			} else if fsEvent.IsDirRemoved() {
				cgEvent = &types.CgroupEvent{
					Path:      fsEvent.Path,
					EventType: types.CgroupEventTypeDelete,
				}
			} else {
				err = &ErrUnexpectFsEvent{fsEvent.RawEvent}
				m.log.Error(err)
				err = nil
				return
			}

			err = m.send(ctx, cgEvent)
			if err != nil {
				return
			}

		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

func (m *Monitor) send(ctx context.Context, cgEvent *types.CgroupEvent) (err error) {
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
	case m.output <- cgEvent:
		m.log.Debugw("Cgroup event sent.",
			"path", path,
		)
	}

	return
}
