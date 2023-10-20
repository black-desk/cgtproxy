package cgfsmon

import (
	"context"

	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	fsevents "github.com/tywkeene/go-fsevents"
)

func (w *CGroupFSMonitor) Events() <-chan types.CGroupEvent {
	return w.events
}

func (w *CGroupFSMonitor) Run(ctx context.Context) (err error) {
	defer Wrap(&err, "running filesystem watcher")
	defer close(w.events)

	ctx, cancel := context.WithCancelCause(ctx)

	err = w.watcher.RegisterEventHandler(&handle{
		ctx: ctx,
		mon: w,
	})
	if err != nil {
		return
	}

	// FIXME(black_desk):
	// This stupid inotify package
	// failed to recursively add a directory
	// when a directory quickly removed
	// after entries in that directory.
	// To fix this.
	// I think we should write an inotify library ourselves.
	err = w.watcher.RecursiveAdd(
		string(w.root),
		fsevents.DirCreatedEvent|fsevents.DirRemovedEvent,
	)
	if err != nil {
		return
	}

	w.log.Info("Going through cgroupfs first time...")
	w.walk(ctx, string(w.root))
	w.log.Info("Going through cgroupfs first time...Done.")

	go func() {
		var err error
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case err = <-w.watcher.Errors:
				w.log.Errorw(
					"Underling filesystem watcher error arrives.",
					"error", err,
				)
			}
		}
	}()

	go func() {
		w.watcher.WatchAndHandle()
		cancel(ErrUnderlingWatcherExited)
	}()

	defer func() {
		var err error
		err = w.watcher.StopAll()
		if err == nil {
			return
		}

		w.log.Errorw(
			"Stopping underling filesystem watcher.",
			"error", err,
		)
	}()

	<-ctx.Done()
	err = ctx.Err()
	if err != nil {
		return
	}

	return
}
