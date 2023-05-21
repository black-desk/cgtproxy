package types

type CgroupEventType uint8

const (
	CgroupEventTypeNew    CgroupEventType = iota // New
	CgroupEventTypeDelete                        // Delete
)

//go:generate stringer -type=CgroupEventType -linecomment

type CgroupEvent struct {
	Path      string
	EventType CgroupEventType
}
