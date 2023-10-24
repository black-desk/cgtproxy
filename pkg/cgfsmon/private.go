package cgfsmon

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/pkg/types"
)

func (m *CGroupFSMonitor) walkFn(events *[]types.CGroupEvent) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			m.log.Debug(
				"Cgroup had been removed.",
				"path", path,
			)
			err = nil
		}

		if err != nil {
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

		path = strings.TrimRight(path, "/")

		if path == string(m.root) {
			return nil
		}

		*events = append(*events, types.CGroupEvent{
			Path:      path,
			EventType: types.CgroupEventTypeNew,
		})

		return nil
	}
}

func (m *CGroupFSMonitor) walk(path string) (ret []types.CGroupEvent, err error) {
	events := []types.CGroupEvent{}

	err = filepath.WalkDir(path, m.walkFn(&events))
	if err != nil {
		return
	}

	ret = events
	return
}

func (m *CGroupFSMonitor) send(ctx context.Context, cgEvents types.CGroupEvents) (err error) {
	for i := range cgEvents.Events {
		path := strings.TrimRight(cgEvents.Events[i].Path, "/")
		cgEvents.Events[i].Path = path
	}

	m.log.Debugw("New cgroup envents.",
		"size", len(cgEvents.Events),
	)

	cnt := len(cgEvents.Events)

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case m.eventsOut <- cgEvents:
		m.log.Debugw("Cgroup events sent.",
			"size", cnt,
		)
	}

	return
}
