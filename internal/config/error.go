package config

import (
	"errors"
	"fmt"

	"github.com/black-desk/cgtproxy/internal/consts"
)

var (
	ErrTooFewPorts              = errors.New("Too few ports for tproxy")
	ErrTooFewMarks              = errors.New("Too few marks for tproxy")
	ErrCannotFoundCgroupv2Mount = errors.New("`cgroup2` mount point not found in /proc/mounts.")
)

type ErrBadRange struct {
	Actual string
}

func (e *ErrBadRange) Error() string {
	return fmt.Sprintf(
		"A `range` must be a string match %s, but we got %s.",
		consts.PortsPattern, e.Actual,
	)
}

type ErrBadProxyName struct {
	Actual string
}

func (e *ErrBadProxyName) Error() string {
	return fmt.Sprintf(
		"Proxy name must not end with -MARK, but we got %s.",
		e.Actual,
	)
}
