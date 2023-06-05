package repeater

import "context"

type Repeater struct {
}

type Opt func(r *Repeater) (ret *Repeater, err error)

func New(opts ...Opt) (ret *Repeater, err error) {
	panic("Unimplemented")
}

func (r *Repeater) Run(ctx context.Context) (err error) {
	panic("Unimplemented")
}
