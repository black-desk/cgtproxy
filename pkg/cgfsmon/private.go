package cgfsmon

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/pkg/types"
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
	case m.eventsOut <- *cgEvent:
		m.log.Debugw("Cgroup event sent.",
			"path", path,
		)
	}

	return
}
