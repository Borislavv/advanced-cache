package cache

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
	"time"
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
	activeEvictorsCh  chan struct{}
	evictsCh          chan int
	evictionThreshold uintptr

	// list from newest to oldest
	shardedOrderedList [sharded.ShardCount]*OrderedList
}

func NewLRU(ctx context.Context, cfg *config.Config) *LRUAlgo {
	lru := &LRUAlgo{
		cfg:               cfg,
		shardedMap:        sharded.NewMap[uint64, *model.Response](cfg.InitStorageLengthPerShard),
		evictionThreshold: uintptr(math.Round(cfg.MemoryLimit * cfg.MemoryFillThreshold)),
		activeEvictorsCh:  make(chan struct{}, maxEvictors),
		evictsCh:          make(chan int, maxEvictors),
	}

	for i := 0; i < sharded.ShardCount; i++ {
		lru.shardedOrderedList[i] = NewOrderedList()
	}

	go lru.logStatInfo(ctx)

	return lru
}

func (c *LRUAlgo) logStatInfo(ctx context.Context) {
	evictsPer3Secs := 0
	t := utils.NewTicker(ctx, time.Second*5)
	for {
		select {
		case <-ctx.Done():
			return
		case evictsPerIter := <-c.evictsCh:
			evictsPer3Secs += evictsPerIter
		case <-t:
			log.Info().Msgf(
				"LRU: evicted %d items at the last 5 seconds. Stat: memory usage: %s, storage len: %d.",
				evictsPer3Secs, utils.FmtMem(c.shardedMap.Mem()), c.shardedMap.Len(),
			)
			evictsPer3Secs = 0
		}
	}
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

	if c.shouldEvict() {
		c.evict(ctx, shardKey)
	}

	c.recordPush(shardKey, resp)
	c.shardedMap.Set(key, resp)
}

func (c *LRUAlgo) Del(req *model.Request) {
	c.shardedMap.Del(req.UniqueKey())
}

func (c *LRUAlgo) shouldEvict() bool {
	return c.shardedMap.Mem() > c.evictionThreshold
}

func (c *LRUAlgo) evict(ctx context.Context, key uint) {
	select {
	case c.activeEvictorsCh <- struct{}{}:
		go func() {
			defer func() { <-c.activeEvictorsCh }()

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

			c.evictsCh <- c.evictBatch(ctx, key, (needEvictMemory/weighPerItem)+1)
		}()
	default:
		// max number of evictors reached
		// skip creation of new one
	}
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
				if resp, found := c.shardedMap.Del(back.Value.(*model.Request).UniqueKey()); found {
					model.RequestsPool.Put(resp.GetRequest())
					model.ResponsePool.Put(resp)
				}
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
