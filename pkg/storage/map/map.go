package sharded

import (
	"context"
	"sync"
	"sync/atomic"
)

const ShardCount uint64 = 4096 // Total number of shards (power of 2 for fast hashing)

// Releasable defines reference-counted resource management for cache values.
type Releasable interface {
	Release() bool
	RefCount() int64
	IncRefCount() int64
	CASRefCount(old, new int64) bool
	StoreRefCount(new int64)
	IsDoomed() bool
	MarkAsDoomed() bool
	ShardListElement() any // Used for pointer to the element in the LRU or similar
}

// Keyer defines a unique key and a precomputed shard key for the value.
type Keyer interface {
	Key() uint64
	ShardKey() uint64
}

// Sizer provides memory usage accounting for cache entries.
type Sizer interface {
	Size() uintptr
}

// Value must implement all cache entry interfaces: keying, sizing, and releasability.
type Value interface {
	Keyer
	Sizer
	Releasable
}

// Map is a sharded concurrent map for high-performance caches.
type Map[V Value] struct {
	shards [ShardCount]*Shard[V]
}

// NewMap creates a new sharded map with preallocated shards and a default per-shard map capacity.
func NewMap[V Value](defaultLen int) *Map[V] {
	m := &Map[V]{}
	for id := uint64(0); id < ShardCount; id++ {
		m.shards[id] = NewShard[V](id, defaultLen)
	}
	return m
}

// MapShardKey calculates the shard index for a given key.
func MapShardKey(key uint64) uint64 {
	return key % ShardCount
}

// Set inserts or updates a value in the correct shard. Returns a releaser for ref counting.
func (smap *Map[V]) Set(value V) *Releaser[V] {
	return smap.shards[value.ShardKey()].Set(value.Key(), value)
}

// Get fetches a value and its releaser from the correct shard.
// found==false means the value is absent.
func (smap *Map[V]) Get(key uint64, shardKey uint64) (value V, releaser *Releaser[V], found bool) {
	return smap.shards[shardKey].Get(key)
}

// Release deletes a value by key, returning how much memory was freed and a pointer to its LRU/list element.
func (smap *Map[V]) Release(key uint64) (freed uintptr, listElem any, ok bool) {
	return smap.Shard(key).Release(key)
}

// Walk applies fn to all key/value pairs in the shard, optionally locking for writing.
func (shard *Shard[V]) Walk(ctx context.Context, fn func(uint64, V), lockWrite bool) {
	if lockWrite {
		shard.Lock()
		defer shard.Unlock()
	} else {
		shard.RLock()
		defer shard.RUnlock()
	}
	for k, v := range shard.items {
		select {
		case <-ctx.Done():
			return
		default:
			fn(k, v)
		}
	}
}

// Shard returns the shard that stores the given key.
func (smap *Map[V]) Shard(key uint64) *Shard[V] {
	return smap.shards[MapShardKey(key)]
}

// WalkShards launches fn concurrently for each shard (key, *Shard[V]).
// The callback runs in a separate goroutine for each shard; fn should be goroutine-safe.
func (smap *Map[V]) WalkShards(fn func(key uint64, shard *Shard[V])) {
	var wg sync.WaitGroup
	wg.Add(int(ShardCount))
	defer wg.Wait()
	for k, s := range smap.shards {
		go func(key uint64, shard *Shard[V]) {
			defer wg.Done()
			fn(key, shard)
		}(uint64(k), s)
	}
}

// Len returns the total number of elements in all shards (O(ShardCount)).
func (smap *Map[V]) Len() int64 {
	var length int64
	for _, shard := range smap.shards {
		length += atomic.LoadInt64(&shard.len)
	}
	return length
}
