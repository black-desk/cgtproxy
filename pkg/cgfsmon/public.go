package cgfsmon

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	fsevents "github.com/tywkeene/go-fsevents"
)

func (w *CGroupFSMonitor) Events() <-chan types.CGroupEvent {
	return w.events
}

func (m *CGroupFSMonitor) walkFn(ctx context.Context) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				m.log.Debug(
					"Cgroup had been removed.",
					"path", path,
				)
				err = nil
			}
			m.log.Errorw(
				"Errors occurred while first time going through cgroupfs.",
				"path", path,
				"error", err,
			)
			err = nil
		}

		if !d.IsDir() {
			return nil
		}

		cgEvent := &types.CGroupEvent{
			Path:      path,
			EventType: types.CgroupEventTypeNew,
		}

		err = m.send(ctx, cgEvent)
		if err != nil {
			return err
		}

		return nil
	}
}

func (m *CGroupFSMonitor) walk(ctx context.Context, path string) {
	err := filepath.WalkDir(path, m.walkFn(ctx))
	if err == nil {
		return
	}

	return
}

func (m *CGroupFSMonitor) send(ctx context.Context, cgEvent *types.CGroupEvent) (err error) {
	path := strings.TrimRight(cgEvent.Path, "/")
	cgEvent.Path = path

	if cgEvent.Path == string(m.root) {
		// NOTE: Ignore cgroup root.
		return nil
	}

	m.log.Debugw("New cgroup envent.",
		"event", cgEvent,
	)

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case m.events <- *cgEvent:
		m.log.Debugw("Cgroup event sent.",
			"path", path,
		)
	}

	return
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
