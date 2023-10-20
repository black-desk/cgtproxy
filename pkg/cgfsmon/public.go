package cgfsmon

import (
	"context"

	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/rjeczalik/notify"
)

func (w *CGroupFSMonitor) Events() <-chan types.CGroupEvent {
	return w.eventsOut
}

func (w *CGroupFSMonitor) Run(ctx context.Context) (err error) {
	defer Wrap(&err, "running filesystem watcher")
	defer close(w.eventsOut)
	defer notify.Stop(w.eventsIn)
	defer close(w.eventsIn)

	err = notify.Watch(string(w.root)+"/...", w.eventsIn, notify.Create, notify.Remove)
	if err != nil {
		return
	}

	w.log.Info("Going through cgroupfs first time...")
	w.walk(ctx, string(w.root))
	w.log.Info("Going through cgroupfs first time...Done.")

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case event := <-w.eventsIn:
			eventType := types.CgroupEventTypeNew
			if event.Event() == notify.InDelete {
				eventType = types.CgroupEventTypeDelete
			}

			err = w.send(ctx, &types.CGroupEvent{
				Path:      event.Path(),
				EventType: eventType,
			})
		}
	}

	<-ctx.Done()
	err = ctx.Err()
	if err != nil {
		return
	}

	return
}
