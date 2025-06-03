package sharded

import (
	"encoding/binary"
	"hash"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

const ShardCount = 2048

type Sizer interface {
	Size() uintptr
}

type (
	Map[K comparable, V Sizer] struct {
		len        *atomic.Int64
		mem        *atomic.Uintptr
		shards     [ShardCount]*shard[K, V]
		hasherPool *sync.Pool // pool of hash/fnv (hash.Hash64)
	}

	shard[K comparable, V Sizer] struct {
		items map[K]V
		sync.RWMutex
	}
)

func NewMap[K comparable, V Sizer](defaultLen int) *Map[K, V] {
	m := &Map[K, V]{
		len: &atomic.Int64{},
		mem: &atomic.Uintptr{},
		hasherPool: &sync.Pool{
			New: func() any { return fnv.New64a() },
		},
	}
	for i := 0; i < ShardCount; i++ {
		m.shards[i] = &shard[K, V]{items: make(map[K]V, defaultLen)}
	}
	return m
}

func (m *Map[K, V]) GetShardKey(key K) uint {
	h := m.hasherPool.Get().(hash.Hash64)
	defer func() {
		h.Reset()
		m.hasherPool.Put(h)
	}()
	_, _ = h.Write(m.key(key))
	return uint(h.Sum64()) % ShardCount
}

// getShardByKey - searches shard by its own key.
func (m *Map[K, V]) getShardByKey(key uint) *shard[K, V] {
	return m.shards[key]
}

// getShard - searches shard by request unique key.
func (m *Map[K, V]) getShard(key K) *shard[K, V] {
	h := m.hasherPool.Get().(hash.Hash64)
	defer func() {
		h.Reset()
		m.hasherPool.Put(h)
	}()
	_, _ = h.Write(m.key(key))
	return m.shards[uint(h.Sum64())%ShardCount]
}

func (m *Map[K, V]) Set(key K, value V) {
	m.len.Add(1)
	m.mem.Add(value.Size())
	s := m.getShard(key)
	s.Lock()
	s.items[key] = value
	s.Unlock()
}

func (m *Map[K, V]) Get(key K, shardKey uint) (value V, found bool) {
	s := m.getShardByKey(shardKey)
	s.RLock()
	v, ok := s.items[key]
	s.RUnlock()
	return v, ok
}

func (m *Map[K, V]) Del(key K) (value V, found bool) {
	s := m.getShard(key)
	s.Lock()
	defer s.Unlock()
	v, f := s.items[key]
	if f {
		delete(s.items, key)
		m.mem.Add(-v.Size())
		m.len.Add(-1)
	}
	return v, f
}

func (m *Map[K, V]) Has(key K) bool {
	s := m.getShard(key)
	s.RLock()
	_, ok := s.items[key]
	s.RUnlock()
	return ok
}

func (m *Map[K, V]) Rng(fn func(K, V)) {
	var wg sync.WaitGroup
	wg.Add(ShardCount)
	for _, s := range m.shards {
		go func(s *shard[K, V]) {
			defer wg.Done()
			defer s.RUnlock()
			s.RLock()
			for k, v := range s.items {
				fn(k, v)
			}
		}(s)
	}
	wg.Wait()
}

func (m *Map[K, V]) Mem() uintptr {
	return m.mem.Load()
}

func (m *Map[K, V]) Len() int64 {
	return m.len.Load()
}

func (m *Map[K, V]) key(key any) []byte {
	switch x := key.(type) {
	case string:
		return []byte(x)
	case int:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(x))
		return buf
	case int64:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(x))
		return buf
	case int32:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(x))
		return buf
	case uint:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(x))
		return buf
	case uintptr:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(x))
		return buf
	case uint64:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, x)
		return buf
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
