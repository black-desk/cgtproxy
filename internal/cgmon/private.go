package cgmon

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/pkg/types"
)

func (m *FSMonitor) walkFn(ctx context.Context) func(path string, d fs.DirEntry, err error) error {
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

func (m *FSMonitor) walk(ctx context.Context, path string) {
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

func (m *FSMonitor) doWalk(ctx context.Context, path string) (err error) {
	err = filepath.WalkDir(path, m.walkFn(ctx))
	return
}

func (m *FSMonitor) send(ctx context.Context, cgEvent *types.CgroupEvent) (err error) {
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
	case m.output <- *cgEvent:
		m.log.Debugw("Cgroup event sent.",
			"path", path,
		)
	}

	return
}
