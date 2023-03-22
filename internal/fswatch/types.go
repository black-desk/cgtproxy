package fswatch

type FsEventType uint8

const (
	FsEventTypeNoOp              FsEventType = iota // NoOp
	FsEventTypePlatformSpecific                     // PlatformSpecific
	FsEventTypeCreated                              // Created
	FsEventTypeUpdated                              // Updated
	FsEventTypeRemoved                              // Removed
	FsEventTypeRenamed                              // Renamed
	FsEventTypeOwnerModified                        // OwnerModified
	FsEventTypeAttributeModified                    // AttributeModified
	FsEventTypeMovedFrom                            // MovedFrom
	FsEventTypeMovedTo                              // MovedTo
	FsEventTypeIsFile                               // IsFile
	FsEventTypeIsDir                                // IsDir
	FsEventTypeIsSymLink                            // IsSymLink
	FsEventTypeLink                                 // Link
	FsEventTypeOverflow                             // Overflow
)

//go:generate stringer -type=FsEventType -linecomment

type FsEvent struct {
	Type FsEventType
	Path string
	Err  error
}

type FsWatcher interface {
	Start() (events <-chan *FsEvent, err error)
}
