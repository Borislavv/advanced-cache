package sharded

import (
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"sync"
	"sync/atomic"
	"unsafe"
)

type (
	Shard[V Keyer] struct {
		*sync.RWMutex
		items        map[uint64]V
		releaserPool *synced.BatchPool[*Releaser[V]]
		id           uint64
		len          int64
		mem          uintptr
	}
	Releaser[V Keyer] struct {
		val   V
		relFn func(v V) bool
		pool  *synced.BatchPool[*Releaser[V]]
	}
)

func NewReleaser[V Keyer](val V, pool *synced.BatchPool[*Releaser[V]]) *Releaser[V] {
	rel := pool.Get()
	*rel = Releaser[V]{
		val:  val,
		pool: pool,
		relFn: func(value V) bool {
			if old := value.RefCount(); value.CASRefCount(old, old-1) {
				if old == 1 && value.IsDoomed() {
					value.Release()
				}
				return true
			}
			return false
		},
	}
	return rel
}

func (r *Releaser[V]) Release() bool {
	if r == nil {
		return true
	}
	ok := r.relFn(r.val)
	r.pool.Put(r)
	return ok
}

func NewShard[V Keyer](id uint64, defaultLen int) *Shard[V] {
	return &Shard[V]{
		RWMutex: &sync.RWMutex{},
		items:   make(map[uint64]V, defaultLen),
		releaserPool: synced.NewBatchPool[*Releaser[V]](synced.PreallocationBatchSize, func() *Releaser[V] {
			return new(Releaser[V])
		}),
		id:  id,
		len: 0,
		mem: 0,
	}
}

func (shard *Shard[V]) ID() uint64 {
	return shard.id
}

func (shard *Shard[V]) Size() uintptr {
	return unsafe.Sizeof(shard) + atomic.LoadUintptr(&shard.mem)
}

func (shard *Shard[V]) Set(key uint64, value V) *Releaser[V] {
	value.StoreRefCount(1)

	shard.Lock()
	shard.items[key] = value
	shard.Unlock()

	atomic.AddInt64(&shard.len, 1)
	atomic.AddUintptr(&shard.mem, value.Size())

	// if someone already could take this item and mark it as doomed (almost impossible but any way)
	return NewReleaser(value, shard.releaserPool)
}

func (shard *Shard[V]) Get(key uint64) (val V, releaser *Releaser[V], isHit bool) {
	shard.RLock()
	value, ok := shard.items[key]
	shard.RUnlock()
	if ok {
		value.IncRefCount()
		return value, NewReleaser(value, shard.releaserPool), true
	}
	return value, nil, false
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
