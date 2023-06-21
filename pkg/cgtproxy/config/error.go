package config

import (
	"errors"
)

var (
	ErrCannotFoundCgroupv2Mount = errors.New("`cgroup2` mount point not found in /proc/mounts.")
)
