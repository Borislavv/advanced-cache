package sharded

import (
	"sync"
	"unsafe"
)

const ShardCount uint64 = 4096

type Sizer interface {
	Size() uintptr
}

type (
	Map[V Sizer] struct {
		shards [ShardCount]*Shard[V]
	}
)

func NewMap[V Sizer](defaultLen int) *Map[V] {
	m := &Map[V]{}
	for id := uint64(0); id < ShardCount; id++ {
		m.shards[id] = NewShard[V](id, defaultLen)
	}
	return m
}

func (smap *Map[V]) GetShardKey(key uint64) uint64 {
	return key % ShardCount
}

func (smap *Map[V]) Set(key uint64, value V) {
	smap.Shard(key).Set(key, value)
}

func (smap *Map[V]) Get(key uint64, shardKey uint64) (value V, found bool) {
	return smap.shards[shardKey].Get(key)
}

func (smap *Map[V]) Del(key uint64) (value V, found bool) {
	return smap.Shard(key).Del(key)
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
	return smap.shards[smap.GetShardKey(key)]
}

func (smap *Map[V]) Walk(fn func(uint64, V), lockWrite bool) {
	var wg sync.WaitGroup
	wg.Add(int(ShardCount))
	defer wg.Wait()
	for key, shard := range smap.shards {
		go func(k int, s *Shard[V]) {
			defer wg.Done()
			shard.Walk(fn, lockWrite)
		}(key, shard)
	}
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

func (smap *Map[V]) Mem() uintptr {
	var mem uintptr
	for _, shard := range smap.shards {
		mem += shard.mem.Load()
	}
	return unsafe.Sizeof(smap) + mem
}

func (smap *Map[V]) Len() int64 {
	var length int64
	for _, shard := range smap.shards {
		length += shard.Len.Load()
	}
	return length
}
