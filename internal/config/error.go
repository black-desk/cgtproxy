package config

import (
	"errors"
)

var (
	ErrTooFewPorts              = errors.New("Too few ports for tproxy")
	ErrTooFewMarks              = errors.New("Too few marks for tproxy")
	ErrCannotFoundCgroupv2Mount = errors.New("`cgroup2` mount point not found in /proc/mounts.")
)
