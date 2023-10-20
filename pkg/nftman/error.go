package nftman

import "errors"

var (
	ErrNftableConnMissing = errors.New("`nftables.Conn` is missing.")
	ErrRerouteMarkMissing = errors.New("reroute mark is missing.")
	ErrLoggerMissing      = errors.New("logger is missing.")
	ErrCGroupRootMissing  = errors.New("cgroupv2 file system mount point is missing.")
)
