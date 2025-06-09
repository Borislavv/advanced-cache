package sharded

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type (
	Shard[V Keyer] struct {
		sync.RWMutex
		id    uint64
		items map[uint64]V
		Len   *atomic.Int64
		mem   *atomic.Uintptr
	}
)

func NewShard[V Keyer](id uint64, defaultLen int) *Shard[V] {
	return &Shard[V]{
		RWMutex: sync.RWMutex{},
		id:      id,
		items:   make(map[uint64]V, defaultLen),
		Len:     &atomic.Int64{},
		mem:     &atomic.Uintptr{},
	}
}

func (shard *Shard[V]) ID() uint64 {
	return shard.id
}

func (shard *Shard[V]) Size() uintptr {
	return unsafe.Sizeof(shard) + shard.mem.Load()
}

func (shard *Shard[V]) Set(key uint64, value V) {
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()

	shard.Len.Add(1)
	shard.mem.Add(value.Size())
}

func (shard *Shard[V]) Get(key uint64) (value V, found bool) {
	shard.RLock()
	v, ok := shard.items[key]
	shard.RUnlock()
	return v, ok
}

func (shard *Shard[V]) Del(key uint64) (value V, found bool) {
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

func (shard *Shard[V]) Release(key uint64) bool {
	shard.Lock()
	v, f := shard.items[key]
	if f {
		delete(shard.items, key)
		shard.mem.Add(-v.Size())
		shard.Len.Add(-1)
		return v.Release()
	}
	shard.Unlock()
	return f
}
