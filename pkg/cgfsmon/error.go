package cgfsmon

import "errors"

var (
	ErrContextMissing     = errors.New("context is missing.")
	ErrCGroupRootNotFound = errors.New("cgroup v2 file system mount point is missing.")
	ErrLoggerMissing      = errors.New("logger is missing.") 
)
