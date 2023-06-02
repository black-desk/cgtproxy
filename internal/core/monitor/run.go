package monitor

import (
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/fsnotify/fsnotify"
)

func (m *Monitor) Run() (err error) {
	defer close(m.output)
	defer Wrap(&err, "Error occurs while running the cgroup monitor.")

	// TODO(black_desk): handle exsiting cgroup

	var cgEvent *types.CgroupEvent
	for fsEvent := range m.watcher.Events {
		if fsEvent.Has(fsnotify.Create) {
			cgEvent = &types.CgroupEvent{
				Path:      fsEvent.Name,
				EventType: types.CgroupEventTypeNew,
			}
		} else if fsEvent.Has(fsnotify.Remove) {
			cgEvent = &types.CgroupEvent{
				Path:      fsEvent.Name,
				EventType: types.CgroupEventTypeDelete,
			}

		} else if fsEvent.Has(fsnotify.Chmod) {
			// We not care about this kind of event
		} else if fsEvent.Has(fsnotify.Write) {
			// We not care about this kind of event
		} else if fsEvent.Has(fsnotify.Rename) {
			// We not care about this kind of event
		} else {
			err = &ErrUnexpectFsEventOp{Op: fsEvent.Op}
			Wrap(&err)
			return
		}

		Log.Infow("New cgroup envent.", "event", cgEvent)

		select {
		case <-m.ctx.Done():
			err = m.ctx.Err()
			return
		case m.output <- cgEvent:
		}
	}
	return
}
