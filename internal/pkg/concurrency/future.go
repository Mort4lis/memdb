package concurrency

type FutureError = Future[error]

type Future[T any] struct {
	valCh <-chan T
}

func NewFuture[T any](valCh <-chan T) Future[T] {
	return Future[T]{valCh: valCh}
}

func (f Future[T]) Get() T {
	return <-f.valCh
}
