package algo

import (
	"container/list"
	"context"
	"fmt"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/rs/zerolog/log"
	"math"
	"sync"
	"sync/atomic"
)

const (
	maxEvictors           = 5
	maxEvictionsPreIter   = 10
	maxEvictionIterations = 10
)

type OrderedList struct {
	mu *sync.Mutex
	*list.List
}

func NewOrderedList() *OrderedList {
	return &OrderedList{
		mu:   new(sync.Mutex),
		List: list.New(),
	}
}

func (l *OrderedList) Size() uintptr {
	return 0
}

type LRUAlgo struct {
	cfg config.Storage

	shardedMap        *sharded.Map[uint64, *model.Response]
	evictionThreshold uintptr
	activeEvictors    *atomic.Int32

	semaCh chan struct{}
	// list from newest to oldest
	shardedOrderedList [sharded.ShardCount]*OrderedList
}

func NewLRU(cfg config.Storage, defaultLen int) *LRUAlgo {
	lru := &LRUAlgo{
		cfg:               cfg,
		shardedMap:        sharded.NewMap[uint64, *model.Response](defaultLen),
		semaCh:            make(chan struct{}, cfg.ParallelEvictionsAvailable),
		evictionThreshold: uintptr(math.Round(cfg.MemoryLimit * cfg.MemoryFillThreshold)),
		activeEvictors:    &atomic.Int32{},
	}

	for i := 0; i < sharded.ShardCount; i++ {
		lru.shardedOrderedList[i] = NewOrderedList()
	}

	return lru
}

func (c *LRUAlgo) Get(req *model.Request) (resp *model.Response, found bool) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)

	resp, found = c.shardedMap.Get(key, shardKey)
	if !found {
		return nil, false
	}

	c.recordHit(shardKey, resp)

	return resp, true
}

func (c *LRUAlgo) Set(ctx context.Context, resp *model.Response) {
	key := resp.GetRequest().UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)

	r, found := c.shardedMap.Get(key, shardKey)
	if found {
		c.recordHit(shardKey, r)
		r.SetMeta(resp.GetMeta())
		return
	}

	if c.isEvictionNecessaryAndAvailable() {
		go c.evict(ctx, shardKey)
	}

	c.recordPush(shardKey, resp)
	c.shardedMap.Set(key, resp)
}

func (c *LRUAlgo) Del(req *model.Request) {
	c.shardedMap.Del(req.UniqueKey())
}

func (c *LRUAlgo) isEvictionNecessaryAndAvailable() bool {
	return c.shardedMap.Mem() > c.evictionThreshold && c.activeEvictors.Load() < maxEvictors
}

func (c *LRUAlgo) evict(ctx context.Context, key uint) {
	c.activeEvictors.Add(1)
	defer c.activeEvictors.Add(-1)

	memDiff := int64(c.shardedMap.Mem() - c.evictionThreshold)
	if memDiff <= 0 {
		return
	}

	memItem := int64(c.shardedMap.Mem()) / c.shardedMap.Len()

	itemsForEviction := int(memDiff/memItem) + (c.cfg.ParallelEvictionsAvailable)

	evicted := c.evictBatch(ctx, key, itemsForEviction)
	_ = evicted

	log.Debug().Msgf("LRU: evicted %d items (mem: %s, len: %d)", evicted, formatBytes(c.shardedMap.Mem()), c.shardedMap.Len())
}

func formatBytes(bytes uintptr) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		t := bytes / TB
		rem := bytes % TB
		return fmt.Sprintf("%dTB %dGB %dMB %dKB %dB", t, rem/GB, (rem%GB)/MB, (rem%MB)/KB, rem%KB)
	case bytes >= GB:
		g := bytes / GB
		rem := bytes % GB
		return fmt.Sprintf("%dGB %dMB %dKB %dB", g, rem/MB, (rem%MB)/KB, rem%KB)
	case bytes >= MB:
		m := bytes / MB
		rem := bytes % MB
		return fmt.Sprintf("%dMB %dKB %dB", m, rem/KB, rem%KB)
	case bytes >= KB:
		k := bytes / KB
		return fmt.Sprintf("%dKB %dB", k, bytes%KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func (c *LRUAlgo) evictBatch(ctx context.Context, shardKey uint, num int) int {
	var evictionsNum int

	s := c.shardedOrderedList[shardKey]
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return evictionsNum
		default:
			if back := s.Back(); evictionsNum < num && back != nil {
				c.shardedMap.Del(back.Value.(*model.Request).UniqueKey())
				s.Remove(back)
				evictionsNum++
			} else {
				return evictionsNum
			}
		}
	}
}

func (c *LRUAlgo) recordHit(shardKey uint, resp *model.Response) {
	resp.Touch()

	el := resp.GetListElement()

	c.moveToFront(shardKey, el)
}

func (c *LRUAlgo) recordPush(shardKey uint, resp *model.Response) {
	resp.Touch()
	resp.SetListElement(c.pushToFront(shardKey, resp.GetRequest()))
}

func (c *LRUAlgo) moveToFront(shardKey uint, el *list.Element) {
	s := c.shardedOrderedList[shardKey]
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MoveToFront(el)
}

func (c *LRUAlgo) pushToFront(shardKey uint, req *model.Request) *list.Element {
	s := c.shardedOrderedList[shardKey]
	s.mu.Lock()
	defer s.mu.Unlock()
	el := s.PushFront(req)
	return el
}
