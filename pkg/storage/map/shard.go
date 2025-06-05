package sharded

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type (
	Shard[K comparable, V Sizer] struct {
		sync.RWMutex
		id    int
		items map[K]V
		Len   *atomic.Int64
		mem   *atomic.Uintptr
	}
)

func NewShard[K comparable, V Sizer](id int, defaultLen int) *Shard[K, V] {
	return &Shard[K, V]{
		RWMutex: sync.RWMutex{},
		id:      id,
		items:   make(map[K]V, defaultLen),
		Len:     &atomic.Int64{},
		mem:     &atomic.Uintptr{},
	}
}

func (shard *Shard[K, V]) ID() int {
	return shard.id
}

func (shard *Shard[K, V]) Size() uintptr {
	return unsafe.Sizeof(shard) + shard.mem.Load()
}

func (shard *Shard[K, V]) Set(key K, value V) {
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()

	shard.Len.Add(1)
	shard.mem.Add(value.Size())
}

func (shard *Shard[K, V]) Get(key K) (value V, found bool) {
	shard.RLock()
	v, ok := shard.items[key]
	shard.RUnlock()
	return v, ok
}

func (shard *Shard[K, V]) Del(key K) (value V, found bool) {
	shard.Lock()
	v, f := shard.items[key]
	if f {
		delete(shard.items, key)
		shard.mem.Add(-v.Size())
		shard.Len.Add(-1)
	}
	shard.Unlock()
	return v, f
}
