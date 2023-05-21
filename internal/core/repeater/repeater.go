package repeater

type Repeater struct {
}

type Opt func(r *Repeater) (ret *Repeater, err error)

func New(opts ...Opt) (ret *Repeater, err error) {
	panic("Unimplemented")
}

func (r *Repeater) Run() (err error) {
	panic("Unimplemented")
}
