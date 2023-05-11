package config

import "errors"

var (
	ErrTooFewPorts       = errors.New("Too few ports for tproxy")
)
