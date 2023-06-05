package monitor

import (
	"context"

	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
)

func (m *Monitor) Run(ctx context.Context) (err error) {
	defer close(m.output)
	defer Wrap(&err, "Error occurs while running the cgroup monitor.")

	// TODO(black_desk): handle exsiting cgroup

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
