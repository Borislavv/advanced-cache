package sharded

import (
	"sync"
	"unsafe"
)

const ShardCount uint64 = 4096

type Keyer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | uintptr
}

type Sizer interface {
	Size() uintptr
}

type (
	Map[K Keyer, V Sizer] struct {
		shards [ShardCount]*Shard[K, V]
	}
)

func NewMap[K Keyer, V Sizer](defaultLen int) *Map[K, V] {
	m := &Map[K, V]{}
	for id := uint64(0); id < ShardCount; id++ {
		m.shards[id] = NewShard[K, V](id, defaultLen)
	}
	return m
}

func (smap *Map[K, V]) GetShardKey(key K) K {
	return key % ShardCount
}

func (smap *Map[K, V]) Set(key K, value V) {
	smap.Shard(key).Set(key, value)
}

func (smap *Map[K, V]) Get(key K, shardKey K) (value V, found bool) {
	return smap.shards[shardKey].Get(key)
}

func (smap *Map[K, V]) Del(key K) (value V, found bool) {
	return smap.Shard(key).Del(key)
}

func (shard *Shard[K, V]) Walk(fn func(K, V), lockWrite bool) {
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

func (smap *Map[K, V]) Shard(key K) *Shard[K, V] {
	return smap.shards[smap.GetShardKey(key)]
}

func (smap *Map[K, V]) Walk(fn func(K, V), lockWrite bool) {
	var wg sync.WaitGroup
	wg.Add(ShardCount)
	defer wg.Wait()
	for key, shard := range smap.shards {
		go func(k int, s *Shard[K, V]) {
			defer wg.Done()
			shard.Walk(fn, lockWrite)
		}(key, shard)
	}
}

func (smap *Map[K, V]) WalkShards(fn func(key int, shard *Shard[K, V])) {
	var wg sync.WaitGroup
	wg.Add(ShardCount)
	defer wg.Wait()
	for k, s := range smap.shards {
		go func(key int, shard *Shard[K, V]) {
			defer wg.Done()
			fn(key, shard)
		}(k, s)
	}
}

func (smap *Map[K, V]) Mem() uintptr {
	var mem uintptr
	for _, shard := range smap.shards {
		mem += shard.mem.Load()
	}
	return unsafe.Sizeof(smap) + mem
}

func (smap *Map[K, V]) Len() int64 {
	var length int64
	for _, shard := range smap.shards {
		length += shard.Len.Load()
	}
	return length
}
