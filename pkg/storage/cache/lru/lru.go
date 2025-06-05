package lru

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"math"
	"runtime"
	"sync"
	"time"
)

const (
	maxEvictors                     = 1
	numberOfPercentsShardsFromEvict = 16
)

var (
	evictsCh chan uintptr
)

type LRU struct {
	ctx context.Context
	cfg *config.Config

	shardedMap        *sharded.Map[uint64, *model.Response]
	activeEvictorsCh  chan struct{}
	evictionThreshold uintptr

	balancer *Balancer
}

func NewLRU(ctx context.Context, cfg *config.Config, balancer *Balancer) *LRU {
	lru := &LRU{
		ctx:               ctx,
		cfg:               cfg,
		shardedMap:        sharded.NewMap[uint64, *model.Response](cfg.InitStorageLengthPerShard),
		evictionThreshold: uintptr(float64(cfg.MemoryLimit) * cfg.MemoryFillThreshold),
		activeEvictorsCh:  make(chan struct{}, maxEvictors),
		balancer:          balancer,
	}

	lru.shardedMap.WalkShards(func(shardKey int, shard *sharded.Shard[uint64, *model.Response]) {
		lru.balancer.register(shard)
	})

	if cfg.IsDebugOn() {
		lru.runLogDebugInfo(ctx)
	}

	return lru
}

func (c *LRU) runLogDebugInfo(ctx context.Context) {
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
					"[lru]: evicted %d (5s), memory usage: %s (limit: %s), storage len: %d, goroutines: %d.",
					evictsPer3Secs, utils.FmtMem(c.shardedMap.Mem()), utils.FmtMem(uintptr(c.cfg.MemoryLimit)), c.shardedMap.Len(), runtime.NumGoroutine(),
				)
				evictsPer3Secs = 0
			}
		}
	}()
}

func (c *LRU) Get(req *model.Request) (resp *model.Response, found bool) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)

	resp, found = c.shardedMap.Get(key, shardKey)
	if !found {
		return nil, false
	}

	c.balancer.move(shardKey, resp.GetListElement())

	if resp.ShouldBeRevalidated() {
		go resp.Revalidate(c.ctx)
	}

	return resp, true
}

func (c *LRU) Set(resp *model.Response) {
	key := resp.GetRequest().UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	resp.SetShardKey(shardKey)
	shard := c.shardedMap.Shard(shardKey)

	r, found := shard.Get(key)
	if found {
		c.balancer.move(shardKey, r.GetListElement())
		r.SetData(resp.GetData())
		return
	}

	respSize := resp.Size()
	if c.shouldEvict(respSize) {
		c.evict(c.ctx, respSize)
	}

	c.balancer.set(resp)

	shard.Set(key, resp)
}

func (c *LRU) Del(req *model.Request) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	c.balancer.remove(key, shardKey)
}

func (c *LRU) shouldEvict(respSize uintptr) bool {
	return c.shardedMap.Mem()+respSize > c.evictionThreshold
}

func (c *LRU) evict(ctx context.Context, respSize uintptr) {
	var memThreshold = c.evictionThreshold

	select {
	case c.activeEvictorsCh <- struct{}{}:
		go func() {
			log.Debug().Msg("[evictor] started")
			defer log.Debug().Msg("[evictor] closed")
			defer func() { <-c.activeEvictorsCh }()

			for {
				log.Info().Msg("[evictor] NEW EVICTOR CYCLE!!!!!!")

				var (
					mem             = c.shardedMap.Mem() + respSize
					length          = uintptr(c.shardedMap.Len())
					needEvictMemory = mem - memThreshold
				)

				if needEvictMemory <= 0 || length == 0 {
					return
				}

				weighPerItem := mem / length
				log.Info().Msgf("weightItem: %d, mem: %d, length: %d", weighPerItem, mem, length)
				if weighPerItem == 0 {
					weighPerItem = 1
				}

				num := needEvictMemory / weighPerItem
				topN := uintptr(sharded.ShardCount / numberOfPercentsShardsFromEvict) // number of top shards by memory usages
				perShard := uintptr(math.Ceil(float64(num) / float64(topN)))          // number of items for eviction per shard
				log.Info().Msgf("----->>> num: %d, topN: %d, perShard: %d", num, topN, perShard)

				log.Info().Msg("new evictors batch")

				wg := sync.WaitGroup{}
				defer wg.Wait()
				for _, n := range c.balancer.mostLoadedList(topN) {
					wg.Add(1)
					go func(n *node) {
						defer wg.Done()
						//if evicted := c.balancer.evictBatch(ctx, node, perShard); evicted > 0 && c.cfg.IsDebugOn() {
						//	evictsCh <- evicted
						//}
						before := n.shard.Len.Load()
						if evicted := c.balancer.evictBatch(ctx, n, perShard); evicted > 0 {
							log.Info().Msgf("%s: --------->>>>>>>> [evicted: %d] topN: %d, perShard: %d, num: %d, shardLen: %d, shardLenBefore: %d, maplen: %d, mapmem: %d, threshold %d",
								time.Now().String(), evicted, topN, perShard, num, n.shard.Len.Load(), before, c.shardedMap.Len(), c.shardedMap.Mem(), c.cfg.MemoryLimit)
							evictsCh <- evicted
						} else {
							log.Info().Msgf("evicted 0, perShard: %d, shardLen: %d, shardID: %d", perShard, n.shard.Len.Load(), n.shard.ID())
						}
					}(n)
				}
			}
		}()
	default:
		// max number of evictors reached
		// skip creation of new one
	}
}
