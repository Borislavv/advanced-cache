package alloc

import synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"

type Batch[T any] struct {
	idx   int64
	maker func() T
	buf   []T
}

func NewBatch[T any](maker func() T) *Batch[T] {
	return &Batch[T]{maker: maker, buf: make([]T, 0, synced.PreallocationBatchSize)}
}

func (b *Batch[T]) Alloc() T {
	return b.buf[b.idx]
}

func (b *Batch[T]) realloc() {

}
