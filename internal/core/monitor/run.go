package monitor

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
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

		cgEvent.Path = strings.TrimRight(cgEvent.Path, "/")

		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		case m.output <- cgEvent:
			Log.Debugw("Cgroup event sent.")
		}

		return nil
	}
}

func (m *Monitor) Run(ctx context.Context) (err error) {
	defer close(m.output)
	defer Wrap(&err, "Error occurs while running the cgroup monitor.")

	Log.Debugw("Initializing cgroup monitor...")

	err = filepath.WalkDir(string(m.root), m.walkFn(ctx))
	if err != nil {
		return
	}

	Log.Debugw("Initializing cgroup monitor done.")

	var cgEvent *types.CgroupEvent
	for {
		select {
		case fsEvent, ok := <-m.watcher.Events:
			if !ok {
				Log.Debugw("Filesystem watcher channel closed.")
				return
			}

			Log.Debugw("New filesystem envent arrived.")

			if fsEvent.IsDirCreated() {
				cgEvent = &types.CgroupEvent{
					Path:      fsEvent.Path,
					EventType: types.CgroupEventTypeNew,
				}
			} else if fsEvent.IsDirRemoved() {
				cgEvent = &types.CgroupEvent{
					Path:      fsEvent.Path,
					EventType: types.CgroupEventTypeDelete,
				}
			} else {
				err = &ErrUnexpectFsEvent{fsEvent.RawEvent}
				return
			}

			cgEvent.Path = strings.TrimRight(cgEvent.Path, "/")

			Log.Debugw("New cgroup envent.", "event", cgEvent)

			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			case m.output <- cgEvent:
				Log.Debugw("Cgroup event sent.")
			}

		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}
