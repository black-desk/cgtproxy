package cgmon

import (
	"context"

	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
)

func (m *FSMonitor) Output() chan<- types.CgroupEvent {
	return m.output
}

func (m *FSMonitor) Run(ctx context.Context) (err error) {
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
