package monitor

import "errors"

var (
	ErrUnexpectFsEventType = errors.New("Unexpected fs event")
	ErrWatcherMissing      = errors.New("Watcher is missing")
	ErrContextMissing      = errors.New("Context is missing")
	ErrOutputMissing       = errors.New("Output is missing")
)
