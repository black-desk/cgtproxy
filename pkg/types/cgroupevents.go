package types

type CGroupEvents struct {
	Events []CGroupEvent
	Result chan<- error
}
