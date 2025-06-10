package sharded

import (
	"sync"
	"sync/atomic"
)

const ShardCount uint64 = 4096

type Releasable interface {
	Release() bool
	RefCount() int64
	IncRefCount() int64
	CASRefCount(old, new int64) bool
	StoreRefCount(new int64)
	IsDoomed() bool
	MarkAsDoomed() bool
	ShardListElement() any
}

type Keyer interface {
	Key() uint64
	ShardKey() uint64
}

type Sizer interface {
	Size() uintptr
}

type Value interface {
	Keyer
	Sizer
	Releasable
}

type (
	Map[V Value] struct {
		shards [ShardCount]*Shard[V]
	}
)

func NewMap[V Value](defaultLen int) *Map[V] {
	m := &Map[V]{}
	for id := uint64(0); id < ShardCount; id++ {
		m.shards[id] = NewShard[V](id, defaultLen)
	}
	return m
}

func MapShardKey(key uint64) uint64 {
	return key % ShardCount
}

func (smap *Map[V]) Set(value V) *Releaser[V] {
	return smap.shards[value.ShardKey()].Set(value.Key(), value)
}

func (smap *Map[V]) Get(key uint64, shardKey uint64) (value V, releaser *Releaser[V], found bool) {
	return smap.shards[shardKey].Get(key)
}

func (smap *Map[V]) Release(key uint64) (freed uintptr, listElem any, ok bool) {
	return smap.Shard(key).Release(key)
}

func (shard *Shard[V]) Walk(fn func(uint64, V), lockWrite bool) {
	if lockWrite {
		shard.Lock()
		defer shard.Unlock()
	} else {
		shard.RLock()
		defer shard.RUnlock()
	}
	for k, v := range shard.items {
		fn(k, v)
	}
}

func (smap *Map[V]) Shard(key uint64) *Shard[V] {
	return smap.shards[MapShardKey(key)]
}

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

func (smap *Map[V]) Len() int64 {
	var length int64
	for _, shard := range smap.shards {
		length += atomic.LoadInt64(&shard.len)
	}
	return length
}
