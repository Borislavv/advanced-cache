package sharded

import (
	"encoding/binary"
	"hash"
	"hash/fnv"
	"sync"
	"unsafe"
)

const ShardCount = 32

type Sizer interface {
	Size() uintptr
}

type (
	Map[K comparable, V Sizer] struct {
		hasherPool  *sync.Pool // pool of hash/fnv (hash.Hash64)
		arrBytePool *sync.Pool // pool of [8]byte
		shards      [ShardCount]*Shard[K, V]
	}
)

func NewMap[K comparable, V Sizer](defaultLen int) *Map[K, V] {
	m := &Map[K, V]{
		hasherPool: &sync.Pool{
			New: func() any { return fnv.New64a() },
		},
		arrBytePool: &sync.Pool{
			New: func() any { return [8]byte{} },
		},
	}
	for id := 0; id < ShardCount; id++ {
		m.shards[id] = NewShard[K, V](id, defaultLen)
	}
	return m
}

func (smap *Map[K, V]) GetShardKey(key K) uint {
	h := smap.hasherPool.Get().(hash.Hash64)
	defer func() {
		h.Reset()
		smap.hasherPool.Put(h)
	}()
	_, _ = h.Write(smap.key(key))
	return uint(h.Sum64()) % ShardCount
}

// getShardByKey - searches Shard by its own key.
func (smap *Map[K, V]) getShardByKey(key uint) *Shard[K, V] {
	return smap.shards[key]
}

// getShard - searches Shard by request unique key.
func (smap *Map[K, V]) getShard(key K) *Shard[K, V] {
	h := smap.hasherPool.Get().(hash.Hash64)
	defer func() {
		h.Reset()
		smap.hasherPool.Put(h)
	}()
	_, _ = h.Write(smap.key(key))
	return smap.shards[uint(h.Sum64())%ShardCount]
}

func (smap *Map[K, V]) Set(key K, value V) {
	smap.getShard(key).Set(key, value)
}

func (smap *Map[K, V]) Get(key K, shardKey uint) (value V, found bool) {
	return smap.getShardByKey(shardKey).Get(key)
}

func (smap *Map[K, V]) Del(key K) (value V, found bool) {
	return smap.getShard(key).Del(key)
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

func (smap *Map[K, V]) Shard(key uint) *Shard[K, V] {
	return smap.shards[key]
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

func (smap *Map[K, V]) key(key any) []byte {
	var buf [8]byte

	buf, ok := smap.arrBytePool.Get().([8]byte)
	if !ok {
		panic("arrBytePool must contains only [8]byte")
	}
	defer smap.arrBytePool.Put(buf)

	bufSl := buf[:]
	switch x := key.(type) {
	case string:
		return []byte(x)
	case int:
		binary.LittleEndian.PutUint64(bufSl, uint64(x))
		return bufSl
	case int64:
		binary.LittleEndian.PutUint64(bufSl, uint64(x))
		return bufSl
	case int32:
		binary.LittleEndian.PutUint64(bufSl, uint64(x))
		return bufSl
	case uint:
		binary.LittleEndian.PutUint64(bufSl, uint64(x))
		return bufSl
	case uintptr:
		binary.LittleEndian.PutUint64(bufSl, uint64(x))
		return bufSl
	case uint64:
		binary.LittleEndian.PutUint64(bufSl, x)
		return bufSl
	case float64:
		panic("float is not available")
	case float32:
		panic("float is not available")
	case bool:
		panic("bool is not available")
	default:
		panic("unsupported key type")
	}
}
