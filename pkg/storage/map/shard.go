package sharded

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type (
	Shard[V Keyer] struct {
		*sync.RWMutex
		items map[uint64]V
		id    uint64
		len   int64
		mem   uintptr
	}
)

func NewShard[V Keyer](id uint64, defaultLen int) *Shard[V] {
	return &Shard[V]{
		RWMutex: &sync.RWMutex{},
		items:   make(map[uint64]V, defaultLen),
		id:      id,
		len:     0,
		mem:     0,
	}
}

func (shard *Shard[V]) ID() uint64 {
	return shard.id
}

func (shard *Shard[V]) Size() uintptr {
	return unsafe.Sizeof(shard) + atomic.LoadUintptr(&shard.mem)
}

func (shard *Shard[V]) Set(key uint64, value V) (release func()) {
	value.StoreRefCount(1)

	shard.Lock()
	shard.items[key] = value
	shard.Unlock()

	atomic.AddInt64(&shard.len, 1)
	atomic.AddUintptr(&shard.mem, value.Size())

	return func() {
		// if someone already could take this item and mark it as doomed (almost impossible but any way)
		if old := value.RefCount(); value.CASRefCount(old, old-1) {
			if old == 1 && value.IsDoomed() {
				value.Release()
			}
		}
	}
}

func (shard *Shard[V]) Get(key uint64) (val V, release func(), isHit bool) {
	shard.RLock()
	value, ok := shard.items[key]
	shard.RUnlock()
	if ok {
		value.IncRefCount()
		return value, func() {
			if old := value.RefCount(); value.CASRefCount(old, old-1) {
				if old == 1 && value.IsDoomed() {
					value.Release()
				}
			}
		}, true
	}
	return value, func() {}, false
}

func (shard *Shard[V]) Release(key uint64) (freed uintptr, listElement any, isHit bool) {
	shard.Lock()
	v, ok := shard.items[key]
	if ok {
		delete(shard.items, key)
		shard.Unlock()

		atomic.AddInt64(&shard.len, -1)
		atomic.AddUintptr(&shard.mem, -v.Size())

		if v.RefCount() == 0 {
			v.Release()
		} else {
			v.MarkAsDoomed()
		}

		return v.Size(), v.ListElement(), true
	}
	shard.Unlock()
	return 0, nil, false
}
