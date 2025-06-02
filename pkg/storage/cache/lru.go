package cache

import (
	"container/list"
	"context"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"hash/maphash"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var maxEvictors = int32(runtime.GOMAXPROCS(0))

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
	cfg config.Config

	shardedMap        *sharded.Map[uint64, *model.Response]
	activeEvictors    *atomic.Int32
	evictionThreshold uintptr

	// list from newest to oldest
	shardedOrderedList [sharded.ShardCount]*OrderedList

	hasherPool *sync.Pool

	evictedCh chan int
}

func NewLRU(cfg config.Config) *LRUAlgo {
	lru := &LRUAlgo{
		cfg:               cfg,
		shardedMap:        sharded.NewMap[uint64, *model.Response](cfg.InitStorageLengthPerShard),
		evictionThreshold: uintptr(math.Round(cfg.MemoryLimit * cfg.MemoryFillThreshold)),
		evictedCh:         make(chan int, maxEvictors),
		activeEvictors:    &atomic.Int32{},
		hasherPool: &sync.Pool{
			New: func() interface{} {
				return &maphash.Hash{}
			},
		},
	}

	for i := 0; i < sharded.ShardCount; i++ {
		lru.shardedOrderedList[i] = NewOrderedList()
	}

	return lru
}

func (c *LRUAlgo) logStatInfo(ctx context.Context) {
	evicted := 0
	ticker := utils.NewTicker(ctx, time.Second*5)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			log.Info().Msgf("[CACHE] LRU: evicted %d keys at the last 5 seconds. "+
				"Memory usage: %s, storage length: %d.", evicted, utils.FmtMemory(c.shardedMap.Mem()), c.shardedMap.Len())
			evicted = 0
		case evictedPerIter := <-c.evictedCh:
			evicted += evictedPerIter
		}
	}
}

func (c *LRUAlgo) Get(ctx context.Context, req *model.Request, fn model.ResponseCreator) (resp *model.Response, isHit bool, err error) {
	h := c.hasherPool.Get().(*maphash.Hash)
	defer c.hasherPool.Put(h)

	key := req.UniqueKey(h)
	shardKey := c.shardedMap.GetShardKey(key)

	resp, found := c.shardedMap.Get(key, shardKey)
	if !found {
		resp, err = c.computeResponse(ctx, req, fn)
		if err != nil {
			return nil, false, errors.New("failed to compute response: " + err.Error())
		}
		c.set(ctx, resp)
		return resp, false, nil
	}

	c.recordHit(shardKey, resp)
	if resp.ShouldBeRevalidated() {
		go resp.Revalidate(ctx)
	}

	return resp, true, nil
}

func (c *LRUAlgo) computeResponse(ctx context.Context, req *model.Request, fn model.ResponseCreator) (*model.Response, error) {
	statusCode, data, header, err := fn(ctx, req)
	if err != nil {
		return nil, err
	}
	return model.NewResponse(
		header, statusCode, req, data, fn,
		c.cfg.RevalidateInterval, c.cfg.RevalidateBeta,
	)
}

func (c *LRUAlgo) set(ctx context.Context, resp *model.Response) {
	h := c.hasherPool.Get().(*maphash.Hash)
	defer c.hasherPool.Put(h)

	key := resp.GetRequest().UniqueKey(h)
	shardKey := c.shardedMap.GetShardKey(key)

	if c.isEvictionNecessaryAndAvailable() {
		go c.evict(ctx, shardKey)
	}

	c.recordPush(shardKey, resp)
	c.shardedMap.Set(key, resp)
}

func (c *LRUAlgo) Del(req *model.Request) {
	h := c.hasherPool.Get().(*maphash.Hash)
	defer c.hasherPool.Put(h)
	c.shardedMap.Del(req.UniqueKey(h))
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
			evictor, evicted, utils.FmtMemory(c.shardedMap.Mem()), c.shardedMap.Len(),
		)
	}

}

func (c *LRUAlgo) evictBatch(ctx context.Context, shardKey uint, num int) int {
	var evictionsNum int

	s := c.shardedOrderedList[shardKey]
	s.mu.Lock()
	defer s.mu.Unlock()

	h := c.hasherPool.Get().(*maphash.Hash)
	defer c.hasherPool.Put(h)

	for {
		select {
		case <-ctx.Done():
			return evictionsNum
		default:
			if back := s.Back(); evictionsNum < num && back != nil {
				c.shardedMap.Del(back.Value.(*model.Request).UniqueKey(h))
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
	el := s.PushFront(req)
	return el
}
