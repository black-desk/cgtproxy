package nftman

import "errors"

var (
	ErrMissingNftableConn = errors.New("`nftables.Conn` is required.")
	ErrMissingRerouteMark = errors.New("Reroute mark is required.")
)
