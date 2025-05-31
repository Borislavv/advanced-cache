package algo

import (
	"container/list"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"math"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	maxEvictors           = 5
	maxEvictionsPreIter   = 10
	maxEvictionIterations = 10
)

type LRUAlgo struct {
	cfg config.Storage

	shardedMap        *sharded.Map[uint64, *model.Response]
	evictionThreshold uintptr
	activeEvictors    *atomic.Int32

	// orderedList mutex and semaphore
	mu     *sync.Mutex
	semaCh chan struct{}
	// list from newest to oldest
	orderedList *list.List

	stringBuildersPool *sync.Pool
}

func NewLRU(cfg config.Storage, defaultLen int) *LRUAlgo {
	return &LRUAlgo{
		cfg:               cfg,
		shardedMap:        sharded.NewMap[uint64, *model.Response](defaultLen),
		evictionThreshold: uintptr(math.Round(cfg.MemoryLimit * cfg.MemoryFillThreshold)),
		mu:                &sync.Mutex{},
		activeEvictors:    &atomic.Int32{},
		semaCh:            make(chan struct{}, cfg.ParallelEvictionsAvailable),
		orderedList:       list.New(),
		stringBuildersPool: &sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
	}
}

func (c *LRUAlgo) Get(req *model.Request) (resp *model.Response, found bool) {
	b := c.stringBuildersPool.Get().(*strings.Builder)
	defer c.stringBuildersPool.Put(b)

	resp, found = c.shardedMap.Get(req.UniqueKey(b))
	if !found {
		return nil, false
	}

	c.recordHit(resp)

	return resp, true
}

func (c *LRUAlgo) Set(ctx context.Context, resp *model.Response) {
	b := c.stringBuildersPool.Get().(*strings.Builder)
	defer c.stringBuildersPool.Put(b)

	key := resp.GetRequest().UniqueKey(b)
	r, found := c.shardedMap.Get(key)
	if found {
		c.recordHit(r)
		r.Copy(resp)
		return
	}

	if c.isEvictionNecessaryAndAvailable() {
		go c.evict(ctx)
	}

	c.recordPush(resp)
	c.shardedMap.Set(key, resp)
}

func (c *LRUAlgo) Del(req *model.Request) {
	b := c.stringBuildersPool.Get().(*strings.Builder)
	defer c.stringBuildersPool.Put(b)
	c.shardedMap.Del(req.UniqueKey(b))
}

func (c *LRUAlgo) isEvictionNecessaryAndAvailable() bool {
	return c.shardedMap.Mem() > c.evictionThreshold && c.activeEvictors.Load() < maxEvictors
}

func (c *LRUAlgo) evict(ctx context.Context) {
	c.activeEvictors.Add(1)
	defer c.activeEvictors.Add(-1)

	evicted := 0
	for i := 0; i < maxEvictionIterations; i++ {
		evicted += c.evictBatch(ctx, maxEvictionsPreIter)
		if c.shardedMap.Mem() < c.evictionThreshold {
			break
		}
	}

	//log.Info().Msgf("LRU: evicted %d items (mem: %dKB, len: %d)\n", evicted, c.shardedMap.Mem()/1024, c.shardedMap.Len())
}

func (c *LRUAlgo) evictBatch(ctx context.Context, num int) int {
	b := c.stringBuildersPool.Get().(*strings.Builder)
	defer c.stringBuildersPool.Put(b)

	var evictionsNum int
loop:
	for {
		select {
		case <-ctx.Done():
			return evictionsNum
		default:
			if back := c.orderedList.Back(); evictionsNum < num && back != nil {
				c.shardedMap.Del(back.Value.(*model.Request).UniqueKey(b))
				c.removeBack(back)
				evictionsNum++
				continue loop
			}
			break loop
		}
	}
	return evictionsNum
}

func (c *LRUAlgo) recordHit(resp *model.Response) {
	resp.Touch()

	el := resp.GetListElement()

	c.moveToFront(el)
}

func (c *LRUAlgo) recordPush(resp *model.Response) {
	resp.Touch()

	el := c.pushToFront(resp.GetRequest())

	resp.SetListElement(el)
}

func (c *LRUAlgo) moveToFront(el *list.Element) {
	c.mu.Lock()
	c.orderedList.MoveToFront(el)
	c.mu.Unlock()
}

func (c *LRUAlgo) pushToFront(req *model.Request) *list.Element {
	c.mu.Lock()
	el := c.orderedList.PushFront(req)
	c.mu.Unlock()
	return el
}

func (c *LRUAlgo) removeBack(back *list.Element) {
	c.mu.Lock()
	c.orderedList.Remove(back)
	c.mu.Unlock()
}
