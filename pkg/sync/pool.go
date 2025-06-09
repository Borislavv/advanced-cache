package synced

import (
	"sync"
	"sync/atomic"
)

const PreallocationBatchSize = 1000

type BatchPool[T any] struct {
	len       int64
	pool      *sync.Pool
	allocFunc func() T
}

func NewBatchPool[T any](preallocateBatchSize int, allocFunc func() T) *BatchPool[T] {
	bp := &BatchPool[T]{allocFunc: allocFunc}
	bp.pool = &sync.Pool{
		New: func() any {
			bp.preallocate(preallocateBatchSize)
			atomic.AddInt64(&bp.len, int64(preallocateBatchSize))
			return bp.pool.Get()
		},
	}
	bp.preallocate(preallocateBatchSize * 10)
	return bp
}

func (bp *BatchPool[T]) Len() int {
	return int(atomic.LoadInt64(&bp.len))
}

func (bp *BatchPool[T]) preallocate(n int) {
	for i := 0; i < n; i++ {
		bp.pool.Put(bp.allocFunc())
	}
}

func (bp *BatchPool[T]) Get() T {
	return bp.pool.Get().(T)
}

func (bp *BatchPool[T]) Put(x T) {
	bp.pool.Put(x)
}
