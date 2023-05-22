package monitor

import (
	"errors"
	"fmt"

	"github.com/fsnotify/fsnotify"
)

var (
	ErrWatcherMissing = errors.New("Watcher is missing.")
	ErrContextMissing = errors.New("Context is missing.")
	ErrOutputMissing  = errors.New("Output is missing.")
)

type ErrUnexpectFsEventOp struct {
	Op fsnotify.Op
}

func (e *ErrUnexpectFsEventOp) Error() string {
	return fmt.Sprintf("Unexpected fs event op: %d \"%s\".", e.Op, e.Op.String())
}
