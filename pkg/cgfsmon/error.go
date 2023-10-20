package cgfsmon

import "errors"

var (
	ErrContextMissing    = errors.New("Context is missing")
	ErrCgroupRootMissing = errors.New("Cgroup root is missing")
)
