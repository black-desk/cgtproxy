package cgfsmon

import "errors"

var (
	ErrContextMissing    = errors.New("Context is missing")
	ErrCGroupRootMissing = errors.New("CGroup root is missing")
)
