package routeman

import "errors"

var (
	ErrNFTManagerMissing      = errors.New("nft manager is missing.")
	ErrConfigMissing          = errors.New("config is missing.")
	ErrCGroupEventChanMissing = errors.New("cgroup event channel is missing.")
)
