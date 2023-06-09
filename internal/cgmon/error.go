package cgmon

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
)

var (
	ErrWatcherMissing    = errors.New("Watcher is missing.")
	ErrOutputMissing     = errors.New("Output is missing.")
	ErrCgroupRootMissing = errors.New("Cgroup root is missing")
)

type ErrUnexpectFsEvent struct {
	*unix.InotifyEvent
}

func (e *ErrUnexpectFsEvent) Error() string {
	return fmt.Sprintf("Unexpected fs event op: %v.", e.InotifyEvent)
}
