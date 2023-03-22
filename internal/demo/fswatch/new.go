package demofswatch

import (
	"context"
	"fmt"
	"sync"

	"github.com/black-desk/deepin-network-proxy-manager/internal/fswatch"
	"github.com/go-playground/validator/v10"
)

type FsWatcherDemo struct {
	path      string
	recursive bool
	events    []string
	ctx       context.Context

	lock    sync.Mutex
	channel chan<- *fswatch.FsEvent
}

type Opt = (func(*FsWatcherDemo) (*FsWatcherDemo, error))

func New(opts ...Opt) (watcher *FsWatcherDemo, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(
			"Failed to create new deepin-network-proxy-manager core: %w",
			err,
		)
	}()

	watcher = &FsWatcherDemo{}
	for i := range opts {
		watcher, err = opts[i](watcher)
		if err != nil {
			watcher = nil
			return
		}
	}

	if err = watcher.initChecks(); err != nil {
		return
	}

	if watcher.ctx == nil {
		watcher.ctx = context.Background()
	}

	return
}

func (w *FsWatcherDemo) initChecks() (err error) {
	type watcherCheck struct {
		Path   string   `validate:"required,dirpath"`
		Events []string `validator:"required,dive,eq=NoOp|eq=PlatformSpecific|eq=Created|eq=Updated|eq=Removed|eq=Renamed|eq=OwnerModified|eq=AttributeModified|eq=MovedFrom|eq=MovedTo|eq=IsFile|eq=IsDir|eq=IsSymLink|eq=Link|eq=Overflow"`
	}

	check := watcherCheck{
		Path:   w.path,
		Events: w.events,
	}

	err = validator.New().Struct(check)
	if err != nil {
		return
	}
	return
}

func WithPath(path string) Opt {
	return func(watcher *FsWatcherDemo) (*FsWatcherDemo, error) {
		watcher.path = path
		return watcher, nil
	}
}

func WithRecursive() Opt {
	return func(watcher *FsWatcherDemo) (*FsWatcherDemo, error) {
		watcher.recursive = true
		return watcher, nil
	}
}

func WithEvents(events []string) Opt {
	return func(watcher *FsWatcherDemo) (*FsWatcherDemo, error) {
		watcher.events = events
		return watcher, nil
	}
}

func WithContext(ctx context.Context) Opt {
	return func(watcher *FsWatcherDemo) (*FsWatcherDemo, error) {
		watcher.ctx = ctx
		return watcher, nil
	}
}
