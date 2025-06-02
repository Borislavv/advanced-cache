package algo

import (
	"container/list"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"math"
	"sync"
	"sync/atomic"
)

const (
	maxEvictors = 64
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
	activeEvictors    *atomic.Int32
	evictionThreshold uintptr

	// list from newest to oldest
	shardedOrderedList [sharded.ShardCount]*OrderedList
}

func NewLRU(cfg config.Storage) *LRUAlgo {
	lru := &LRUAlgo{
		cfg:               cfg,
		shardedMap:        sharded.NewMap[uint64, *model.Response](cfg.InitStorageLengthPerShard),
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

	if resp.ShouldBeRevalidated() {
		go resp.Revalidate()
	}

	return resp, true
}

func (c *LRUAlgo) Set(ctx context.Context, resp *model.Response) {
	key := resp.GetRequest().UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)

	r, found := c.shardedMap.Get(key, shardKey)
	if found {
		c.recordHit(shardKey, r)
		r.SetDatum(resp.GetDatum())
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
	evictor := c.activeEvictors
	defer c.activeEvictors.Add(-1)

	if needEvictMemory := int(c.shardedMap.Mem() - c.evictionThreshold); needEvictMemory > 0 {
		weighPerItem := int(c.shardedMap.Mem()) / int(c.shardedMap.Len())
		numberOfItemsForEviction := needEvictMemory/weighPerItem + 1

		evicted := c.evictBatch(ctx, key, numberOfItemsForEviction)

		log.Debug().Msgf(
			"[EVICTOR: #%d] LRU: evicted %d items (mem: %s, len: %d)",
			evictor, evicted, utils.FmtMem(c.shardedMap.Mem()), c.shardedMap.Len(),
		)
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
	c.moveToFront(shardKey, resp.GetListElement())
}

func (c *LRUAlgo) recordPush(shardKey uint, resp *model.Response) {
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
	return s.PushFront(req)
}
