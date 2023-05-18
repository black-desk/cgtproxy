package pool

import "sync"

type Pool struct {
	p       *sync.Pool
	onPutFn func(x any) any
}

func New(newFn func() any, onPutFn func(x any) any) (ret *Pool) {
	ret = &Pool{
		p:       &sync.Pool{New: newFn},
		onPutFn: onPutFn,
	}
	return
}

func (p *Pool) Get() any {
	return p.p.Get()
}

func (p *Pool) Put(x any) {
	if p.onPutFn == nil {
		p.p.Put(x)
		return
	}
	p.p.Put(p.onPutFn(x))
}
