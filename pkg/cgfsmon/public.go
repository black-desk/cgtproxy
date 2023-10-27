package cgfsmon

import (
	"context"

	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/rjeczalik/notify"
)

func (w *CGroupFSMonitor) Events() <-chan types.CGroupEvents {
	return w.eventsOut
}

func (w *CGroupFSMonitor) RunCGroupMonitor(ctx context.Context) (err error) {
	defer Wrap(&err, "running filesystem watcher")
	defer close(w.eventsOut)
	defer notify.Stop(w.eventsIn)
	defer close(w.eventsIn)

	err = notify.Watch(string(w.root)+"/...", w.eventsIn, notify.Create, notify.Remove)
	if err != nil {
		return
	}

	w.log.Info("Going through cgroupfs first time...")
	var events types.CGroupEvents
	events.Events, err = w.walk(string(w.root))
	w.log.Info("Going through cgroupfs first time...Done.")
	err = w.send(ctx, events)
	if err != nil {
		return
	}

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case eventInfo := <-w.eventsIn:

			event := eventInfo.Event()
			path := eventInfo.Path()

			w.log.Debugw(
				"New filesystem notify arrived.",
				"event", event.String(),
				"path", path,
			)
			eventType := types.CgroupEventTypeNew
			if event == notify.Remove {
				eventType = types.CgroupEventTypeDelete
			}

			err = w.send(ctx, types.CGroupEvents{Events: []types.CGroupEvent{{
				Path:      path,
				EventType: eventType,
			}}})
		}
	}

	<-ctx.Done()
	err = context.Cause(ctx)
	if err != nil {
		return
	}

	return
}
