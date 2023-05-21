package watcher

func (w *Watcher) Run() (err error) {
	<-w.ctx.Done()
	return w.Close()
}
