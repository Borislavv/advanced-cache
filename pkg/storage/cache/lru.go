package cache

import (
	"container/list"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

const (
	maxEvictors = 12
)

var (
	evictsCh chan uintptr
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
	ctx context.Context
	cfg *config.Config

	shardedMap        *sharded.Map[uint64, *model.Response]
	activeEvictorsCh  chan struct{}
	evictionThreshold uintptr

	// list from newest to oldest
	shardedOrderedList [sharded.ShardCount]*OrderedList
}

func NewLRU(ctx context.Context, cfg *config.Config) *LRUAlgo {
	lru := &LRUAlgo{
		ctx:               ctx,
		cfg:               cfg,
		shardedMap:        sharded.NewMap[uint64, *model.Response](cfg.InitStorageLengthPerShard),
		evictionThreshold: uintptr(float64(cfg.MemoryLimit) * cfg.MemoryFillThreshold),
		activeEvictorsCh:  make(chan struct{}, maxEvictors),
	}

	for i := 0; i < sharded.ShardCount; i++ {
		lru.shardedOrderedList[i] = NewOrderedList()
	}

	if cfg.IsDebugOn() {
		lru.runLogDebugInfo(ctx)
	}

	return lru
}

func (c *LRUAlgo) runLogDebugInfo(ctx context.Context) {
	evictsCh = make(chan uintptr, maxEvictors)

	go func() {
		var evictsPer3Secs uintptr
		t := utils.NewTicker(ctx, time.Second*5)
		for {
			select {
			case <-ctx.Done():
				return
			case evictsPerIter := <-evictsCh:
				evictsPer3Secs += evictsPerIter
			case <-t:
				log.Debug().Msgf(
					"[lru]: evicted %d (5s), memory usage: %s (limit: %s), storage len: %d.",
					evictsPer3Secs, utils.FmtMem(c.shardedMap.Mem()), utils.FmtMem(uintptr(c.cfg.MemoryLimit)), c.shardedMap.Len(),
				)
				evictsPer3Secs = 0
			}
		}
	}()
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
		go resp.Revalidate(c.ctx)
	}

	return resp, true
}

func (c *LRUAlgo) Set(resp *model.Response) {
	key := resp.GetRequest().UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)

	r, found := c.shardedMap.Get(key, shardKey)
	if found {
		c.recordHit(shardKey, r)
		r.SetData(resp.GetData())
		return
	}

	if c.shouldEvict() {
		c.evict(c.ctx, shardKey)
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
			defer log.Debug().Msg("[evictor] closed")
			log.Debug().Msg("[evictor] spawned")

			for {
				var (
					mem             = c.shardedMap.Mem()
					length          = uintptr(c.shardedMap.Len())
					memThreshold    = c.evictionThreshold
					needEvictMemory = mem - memThreshold
				)

				if needEvictMemory <= 0 || length == 0 {
					return
				}

				weighPerItem := mem / length
				if weighPerItem == 0 {
					weighPerItem = 1
				}

				const coefficient float64 = 1.25
				evicted := c.evictBatch(ctx, key, uintptr(float64(needEvictMemory/weighPerItem)*coefficient))

				log.Info().Msgf("evicted %d, but must %s", evicted, utils.FmtMem(uintptr(float64(needEvictMemory/weighPerItem)*coefficient)))
				if c.cfg.IsDebugOn() {
					evictsCh <- evicted
				}
			}
		}()
	default:
		// max number of evictors reached
		// skip creation of new one
	}
}

func (c *LRUAlgo) evictOne(shardKey uint) {
	shard := c.shardedOrderedList[shardKey]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if back := shard.Back(); back != nil {
		defer shard.Remove(back)
		if resp, found := c.shardedMap.Del(back.Value.(*model.Request).UniqueKey()); found {
			model.RequestsPool.Put(resp.GetRequest())
			model.ResponsePool.Put(resp)
		}
	}
}

func (c *LRUAlgo) evictBatch(ctx context.Context, shardKey uint, num uintptr) uintptr {
	var evictionsNum uintptr

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
					evictionsNum++
				} else {
					log.Info().Msg("[evictor] resp not found")
				}
				shard.Remove(back)
			} else {
				log.Info().Msgf("[evictor] back is nil: %v or evictionsNum < num: %v", back == nil, evictionsNum < num)
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
