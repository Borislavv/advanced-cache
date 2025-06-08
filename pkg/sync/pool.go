package synced

import (
	"sync"
	"sync/atomic"
)

const PreallocationBatchSize = 100

type BatchPool[T any] struct {
	len       *atomic.Int64
	pool      *sync.Pool
	allocFunc func() T
}

func NewBatchPool[T any](preallocateBatchSize int, allocFunc func() T) *BatchPool[T] {
	bp := &BatchPool[T]{allocFunc: allocFunc, len: &atomic.Int64{}}
	bp.pool = &sync.Pool{
		New: func() any {
			bp.len.Add(int64(preallocateBatchSize))
			bp.preallocate(preallocateBatchSize)
			return bp.pool.Get()
		},
	}
	bp.preallocate(preallocateBatchSize * 10)
	return bp
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

func (bp *BatchPool[T]) Len() int {
	return int(bp.len.Load())
}
