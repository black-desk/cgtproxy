package cgtproxy

import (
	"errors"
)

var (
	ErrConfigMissing        = errors.New("config is missing.")
	ErrLoggerMissing        = errors.New("logger is missing.")
	ErrCGroupMonitorMissing = errors.New("cgroup monitor is missing.")
	ErrRouteManagerMissing  = errors.New("route manager is missing.")
)
