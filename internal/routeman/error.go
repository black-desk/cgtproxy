package routeman

import "errors"

var (
	ErrTableMissing           = errors.New("Table is missing")
	ErrConfigMissing          = errors.New("Config is missing")
	ErrCgroupEventChanMissing = errors.New("CgroupEvent chan is missing")
)
