package watcher

import "errors"

var (
	ErrContextMissing = errors.New("Context is missing")
	ErrConfigMissing  = errors.New("Config is missing")
)
