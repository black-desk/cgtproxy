package cgfsmon

import (
	"os"
	"strconv"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/rjeczalik/notify"
	"go.uber.org/zap"
)

type CGroupFSMonitor struct {
	eventsOut chan types.CGroupEvents
	eventsIn  chan notify.EventInfo
	root      config.CGroupRoot
	log       *zap.SugaredLogger
}

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/pkg/cgfsmon.CGroupFSMonitor -as interfaces.CGroupMonitor -o ../interfaces/cgmon.go

func New(opts ...Opt) (ret *CGroupFSMonitor, err error) {
	defer Wrap(&err, "create filesystem watcher")

	w := &CGroupFSMonitor{}

	w.eventsOut = make(chan types.CGroupEvents)

	for i := range opts {
		w, err = opts[i](w)
		if err != nil {
			return
		}
	}

	if w.log == nil {
		w.log = zap.NewNop().Sugar()
	}

	if w.root == "" {
		err = ErrCGroupRootNotFound
		return
	}

	// FIXME:
	// github.com/rjeczalik/notify may drop events when the receiver is too slow
	// Related issues:
	// - https://github.com/rjeczalik/notify/issues/85
	// - https://github.com/rjeczalik/notify/issues/98
	//
	// To mitigate this issue, we use a buffered channel to receive events.
	// The buffer size can be configured via CGTPROXY_MONITOR_BUFFER_SIZE environment variable.
	// A larger buffer can handle more events in a short period but consumes more memory.
	// If you notice event loss, try increasing this value.

	// Get buffer size from environment variable, default to 1024
	bufferSize := 1024
	if envSize := os.Getenv("CGTPROXY_MONITOR_BUFFER_SIZE"); envSize != "" {
		size, err := strconv.Atoi(envSize)
		if err != nil {
			w.log.Warnw("Invalid buffer size in CGTPROXY_MONITOR_BUFFER_SIZE, using default",
				"value", envSize,
				"error", err,
			)
		} else if size < 1 {
			w.log.Warnw("Buffer size must be greater than 0, using default",
				"value", size,
			)
		} else {
			bufferSize = size
		}
	}

	w.eventsIn = make(chan notify.EventInfo, bufferSize)

	ret = w

	w.log.Debugw("Create a cgroupv2 filesystem monitor.",
		"buffer_size", bufferSize,
	)

	return
}

type Opt func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error)

func WithCgroupRoot(root config.CGroupRoot) Opt {
	return func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error) {
		if root == "" {
			err = ErrCGroupRootNotFound
			return
		}

		w.root = root
		ret = w
		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(w *CGroupFSMonitor) (ret *CGroupFSMonitor, err error) {
		if log == nil {
			err = ErrLoggerMissing
			return
		}

		w.log = log
		ret = w
		return
	}
}
