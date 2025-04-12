package concurrency

type PromiseError = Promise[error]

type Promise[T any] struct {
	promised bool
	valCh    chan T
}

func NewPromise[T any]() Promise[T] {
	return Promise[T]{valCh: make(chan T, 1)}
}

func (p *Promise[T]) Set(v T) {
	if p.promised {
		return
	}

	p.promised = true
	p.valCh <- v
	close(p.valCh)
}

func (p *Promise[T]) Future() Future[T] {
	return NewFuture(p.valCh)
}
