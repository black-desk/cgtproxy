package config

import (
	"errors"
)

var (
	ErrCannotFoundCgroupv2Root = errors.New("`cgroup2` mount point not found.")
)
