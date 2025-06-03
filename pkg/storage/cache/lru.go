package cache

import (
	"container/list"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/rs/zerolog/log"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	maxEvictors = 12
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

type LRUAlgo struct {
	cfg *config.Config

	shardedMap        *sharded.Map[uint64, *model.Response]
	activeEvictors    *atomic.Int32
	evictionThreshold uintptr

	// list from newest to oldest
	shardedOrderedList [sharded.ShardCount]*OrderedList
}

func NewLRU(ctx context.Context, cfg *config.Config) *LRUAlgo {
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

func (c *LRUAlgo) Get(ctx context.Context, req *model.Request) (resp *model.Response, found bool) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)

	resp, found = c.shardedMap.Get(key, shardKey)
	if !found {
		return nil, false
	}
	c.recordHit(shardKey, resp)

	if resp.ShouldBeRevalidated() {
		go resp.Revalidate(ctx)
	}

	return resp, true
}

func (c *LRUAlgo) Set(ctx context.Context, resp *model.Response) {
	key := resp.GetRequest().UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)

	r, found := c.shardedMap.Get(key, shardKey)
	if found {
		c.recordHit(shardKey, r)
		r.SetData(resp.GetData())
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

	mem := int(c.shardedMap.Mem())
	length := int(c.shardedMap.Len())

	needEvictMemory := mem - int(c.evictionThreshold)
	if needEvictMemory <= 0 || length == 0 {
		return
	}

	weighPerItem := mem / length
	if weighPerItem == 0 {
		weighPerItem = 1
	}

	evicted := c.evictBatch(ctx, key, (needEvictMemory/weighPerItem)+1)

	log.Debug().Msg("evicted " + strconv.Itoa(evicted) + " items (mem: " +
		strconv.Itoa(int(c.shardedMap.Mem())) + " bytes, len: " + strconv.Itoa(int(c.shardedMap.Len())) + ")")
}

func (c *LRUAlgo) evictBatch(ctx context.Context, shardKey uint, num int) int {
	var evictionsNum int

	shard := c.shardedOrderedList[shardKey]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return evictionsNum
		default:
			if back := shard.Back(); evictionsNum < num && back != nil {
				c.shardedMap.Del(back.Value.(*model.Request).UniqueKey())
				shard.Remove(back)
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
	shard := c.shardedOrderedList[shardKey]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.MoveToFront(el)
}

func (c *LRUAlgo) pushToFront(shardKey uint, req *model.Request) *list.Element {
	shard := c.shardedOrderedList[shardKey]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	return shard.PushFront(req)
}
