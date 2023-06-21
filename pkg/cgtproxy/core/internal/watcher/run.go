package watcher

import (
	"context"
	. "github.com/black-desk/lib/go/errwrap"
	fsevents "github.com/tywkeene/go-fsevents"
)

func (w *Watcher) Run(ctx context.Context) (err error) {
	defer Wrap(&err, "Error occurs while running the filesystem watcher.")

	// FIXME(black_desk):
	// This stupid inotify package
	// failed to recursively add a directory
	// when a directory quickly removed
	// after entries in that directory.
	// To fix this.
	// I think we should write an inotify library ourselves.
	err = w.RecursiveAdd(
		string(w.root),
		fsevents.DirCreatedEvent|fsevents.DirRemovedEvent,
	)
	if err != nil {
		return
	}

	go w.WatchAndHandle()

	<-ctx.Done()
	err = w.StopAll()
	if err != nil {
		return
	}

	err = ctx.Err()
	if err != nil {
		return
	}

	return
}
