package config

import (
	"errors"
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
)

var (
	ErrTooFewPorts              = errors.New("Too few ports for tproxy")
	ErrCannotFoundCgroupv2Mount = errors.New("`cgroup2` mount point not found in /proc/mounts.")
)

type ErrWrongPortsPattern struct {
	Actual string
}

func (e *ErrWrongPortsPattern) Error() string {
	return fmt.Sprintf(
		"`tproxy-ports` must be range string match %s, but we got %s",
		consts.PortsPattern, e.Actual,
	)
}
