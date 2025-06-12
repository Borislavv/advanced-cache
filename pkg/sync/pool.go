package synced

import (
	"sync"
	"sync/atomic"
)

// PreallocationBatchSize is a sensible default for how many objects are preallocated in a single batch.
// It's used for both initial and on-demand pool growth.
const PreallocationBatchSize = 1000

// BatchPool is a high-throughput generic object pool with batch preallocation.
//
// The main goal is to:
// - Minimize allocations by reusing objects.
// - Reduce allocation spikes by preallocating objects in large batches.
// - Provide simple Get/Put API similar to sync.Pool but with better bulk allocation behavior.
type BatchPool[T any] struct {
	len       int64      // Current number of objects ever preallocated (approximate)
	pool      *sync.Pool // Underlying sync.Pool for thread-safe pooling
	allocFunc func() T   // Function to create new T
}

// NewBatchPool creates a new BatchPool with an initial preallocation.
// - preallocateBatchSize: how many objects to add to the pool per allocation batch.
// - allocFunc: function to construct a new T.
func NewBatchPool[T any](preallocateBatchSize int, allocFunc func() T) *BatchPool[T] {
	bp := &BatchPool[T]{allocFunc: allocFunc}
	bp.pool = &sync.Pool{
		New: func() any {
			// When sync.Pool is empty, preallocate a new batch.
			bp.preallocate(preallocateBatchSize)
			atomic.AddInt64(&bp.len, int64(preallocateBatchSize))
			// Return one object from the freshly preallocated batch.
			return bp.pool.Get()
		},
	}
	// Initial larger preallocation for "warm start".
	bp.preallocate(preallocateBatchSize * 10)
	return bp
}

// Len returns the number of objects ever preallocated (does not track live/used count).
func (bp *BatchPool[T]) Len() int {
	return int(atomic.LoadInt64(&bp.len))
}

// preallocate fills the pool with n new objects using allocFunc.
func (bp *BatchPool[T]) preallocate(n int) {
	for i := 0; i < n; i++ {
		bp.pool.Put(bp.allocFunc())
	}
}

// Get retrieves an object from the pool, allocating if necessary.
// Never returns nil (unless allocFunc does).
func (bp *BatchPool[T]) Get() T {
	return bp.pool.Get().(T)
}

// Put returns an object to the pool for future reuse.
func (bp *BatchPool[T]) Put(x T) {
	bp.pool.Put(x)
}
