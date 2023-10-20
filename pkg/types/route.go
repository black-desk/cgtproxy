package types

type TargetOp uint32

const (
	TargetNoop   TargetOp = iota // noop
	TargetDrop                   // drop
	TargetTProxy                 //tproxy
	TargetDirect                 //direct
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=TargetOp -linecomment

type Target struct {
	Op    TargetOp
	Chain string
}

type Route struct {
	Path   string
	Target Target
}
