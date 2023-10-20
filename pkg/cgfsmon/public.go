package cgfsmon

import (
	"context"

	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	fsevents "github.com/tywkeene/go-fsevents"
)

func (w *CGroupFSMonitor) Events() <-chan types.CgroupEvent {
	return w.events
}

func (w *CGroupFSMonitor) Run(ctx context.Context) (err error) {
	defer Wrap(&err, "running filesystem watcher.")

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

	go w.watcher.WatchAndHandle()

	<-ctx.Done()
	err = w.watcher.StopAll()
	if err != nil {
		return
	}

	err = ctx.Err()
	if err != nil {
		return
	}

	return
}
