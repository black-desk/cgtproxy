package types

type CgroupEventType uint8

const (
	CgroupEventTypeNew    CgroupEventType = iota // New
	CgroupEventTypeDelete                        // Delete
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=CgroupEventType -linecomment

type CGroupEvent struct {
	Path      string
	EventType CgroupEventType
	Result    chan<- error
}
