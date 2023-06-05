package watcher

import (
	"context"

	. "github.com/black-desk/lib/go/errwrap"
	fsevents "github.com/tywkeene/go-fsevents"
)

func (w *Watcher) Run(ctx context.Context) (err error) {
	defer Wrap(&err, "Error occurs while running the filesystem watcher.")

	err = w.RecursiveAdd(
		string(w.root),
		fsevents.DirCreatedEvent|fsevents.DirRemovedEvent,
	)
	if err != nil {
		return
	}

	go w.Watch()

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
