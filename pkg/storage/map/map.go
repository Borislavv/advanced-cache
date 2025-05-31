package sharded

import (
	"encoding/binary"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

const shardCount = 1024

type Sizer interface {
	Size() uintptr
}

type (
	Map[K comparable, V Sizer] struct {
		mem    *atomic.Uintptr
		len    *atomic.Int64
		shards [shardCount]*shard[K, V]
	}

	shard[K comparable, V Sizer] struct {
		items map[K]V
		sync.RWMutex
	}
)

func NewMap[K comparable, V Sizer](defaultLen int) *Map[K, V] {
	m := &Map[K, V]{mem: &atomic.Uintptr{}, len: &atomic.Int64{}}
	for i := 0; i < shardCount; i++ {
		m.shards[i] = &shard[K, V]{items: make(map[K]V, defaultLen)}
	}
	return m
}

func (m *Map[K, V]) getShard(key K) *shard[K, V] {
	hash := fnv.New32a()
	_, _ = hash.Write(m.key(key))
	return m.shards[uint(hash.Sum32())%shardCount]
}

func (m *Map[K, V]) Set(key K, value V) {
	m.len.Add(1)
	m.mem.Add(value.Size())
	s := m.getShard(key)
	s.Lock()
	s.items[key] = value
	s.Unlock()
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	s := m.getShard(key)
	s.RLock()
	v, ok := s.items[key]
	s.RUnlock()
	return v, ok
}

func (m *Map[K, V]) Del(key K) {
	s := m.getShard(key)
	s.Lock()
	defer s.Unlock()
	v, f := s.items[key]
	if f {
		delete(s.items, key)
		m.mem.Add(-v.Size())
		m.len.Add(-1)
	}
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
	wg.Add(shardCount)
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
	case uintptr:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(x))
		return buf
	default:
		panic("unsupported key type")
	}
}
