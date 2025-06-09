package lru

import (
	"context"
	"runtime"
	"time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
)

var (
	maxEvictors    = runtime.GOMAXPROCS(0)
	evictionStatCh = make(chan evictionStat, maxEvictors)
)

type evictionStat struct {
	items int
	mem   uintptr
}

type LRU struct {
	ctx               context.Context
	cfg               *config.Config
	shardedMap        *sharded.Map[*model.Response]
	balancer          *Balancer
	evictionThreshold uintptr
	evictionTriggerCh chan struct{}
}

func NewLRU(ctx context.Context, cfg *config.Config) *LRU {
	lru := &LRU{
		ctx:               ctx,
		cfg:               cfg,
		shardedMap:        sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard),
		evictionThreshold: uintptr(float64(cfg.MemoryLimit) * cfg.MemoryFillThreshold),
		evictionTriggerCh: make(chan struct{}, maxEvictors),
	}

	lru.balancer = NewBalancer(lru.shardedMap)
	lru.shardedMap.WalkShards(func(shardKey uint64, shard *sharded.Shard[*model.Response]) {
		lru.balancer.register(shard)
	})

	lru.spawnEvictors()

	if cfg.IsDebugOn() {
		lru.runLogDebugInfo()
	}

	return lru
}

func (c *LRU) Get(req *model.Request) (resp *model.Response, release func(), isHit bool) {
	var (
		key      = req.Key()
		shardKey = req.ShardKey()
	)

	resp, release, found := c.shardedMap.Get(key, shardKey)
	if found {
		c.onFound(shardKey, resp, nil)
		return resp, release, true
	}

	return nil, release, false
}

func (c *LRU) onFound(shardKey uint64, target *model.Response, source *model.Response) {
	if source != nil {
		target.SetData(source.GetData())
	}

	c.balancer.move(shardKey, target.GetListElement())

	if target.ShouldBeRevalidated(source) {
		go target.Revalidate(c.ctx)
	}
}

func (c *LRU) Set(newResp *model.Response) (release func()) {
	var (
		key      = newResp.GetRequest().Key()
		shardKey = newResp.GetRequest().ShardKey()
		shard    = c.shardedMap.Shard(shardKey)
	)

	resp, release, found := shard.Get(key)
	if found {
		c.onFound(shardKey, resp, newResp)
		return release
	}

	c.onSet(key, shard, newResp)

	return release
}

func (c *LRU) onSet(key uint64, shard *sharded.Shard[*model.Response], resp *model.Response) {
	if c.shouldEvict(resp.Size()) {
		c.evict()
	}

	shard.Set(key, resp)

	c.balancer.rebalance(c.balancer.set(resp))
}

func (c *LRU) del(req *model.Request) (freedMem uintptr, isHit bool) {
	return c.balancer.remove(req.Key(), req.ShardKey())
}

func (c *LRU) spawnEvictors() {
	for i := 0; i < maxEvictors; i++ {
		c.spawnEvictor()
	}
}

func (c *LRU) spawnEvictor() {
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-c.evictionTriggerCh:
				num, memory := c.evictMem()
				if c.cfg.IsDebugOn() {
					if num > 0 || memory > 0 {
						evictionStatCh <- evictionStat{
							items: num,
							mem:   memory,
						}
					}
				}
			}
		}
	}()
}

func (c *LRU) shouldEvict(respSize uintptr) bool {
	return c.shardedMap.Mem()+respSize > c.evictionThreshold
}

func (c *LRU) evict() {
	select {
	case c.evictionTriggerCh <- struct{}{}:
	default:
		// if async evictor cannot process your request
		//otherwise use a manual approach
		items, memory := c.evictMem()
		if c.cfg.IsDebugOn() {
			if items > 0 || memory > 0 {
				evictionStatCh <- evictionStat{
					items: items,
					mem:   memory,
				}
			}
		}
	}
}

func (c *LRU) evictMem() (items int, mem uintptr) {
	const (
		evictItemsPerIter   = 75
		topPercentageShards = 25
		maxEvictIterations  = 15
	)
	for iter := 0; iter < maxEvictIterations; iter++ {
		for _, node := range c.balancer.mostLoadedList(topPercentageShards) {
			back := node.lruList.Back()
			if back == nil {
				c.balancer.rebalance(node)
				continue
			}
			if freedMem, found := c.del(back.Value); found {
				items++
				mem += freedMem
			}
			if items >= evictItemsPerIter {
				return items, mem
			}
		}
	}
	return items, mem
}

func (c *LRU) runLogDebugInfo() {
	go func() {
		var (
			evictsNumPer5Sec int
			evictsMemPer5Sec uintptr
		)
		ticker := utils.NewTicker(c.ctx, 5*time.Second)
		for {
			select {
			case <-c.ctx.Done():
				return
			case evictionStatModel := <-evictionStatCh:
				evictsNumPer5Sec += evictionStatModel.items
				evictsMemPer5Sec += evictionStatModel.mem
			case <-ticker:
				log.Debug().Msgf(
					"[lru]: evicted [n: %d, mem: %dbytes] (5s), "+
						"memory usage: %sbytes (limit: %sbytes), "+
						"storage len: %d, goroutines: %d.",
					evictsNumPer5Sec,
					evictsMemPer5Sec,
					utils.FmtMem(c.shardedMap.Mem()),
					utils.FmtMem(uintptr(c.cfg.MemoryLimit)),
					c.shardedMap.Len(),
					runtime.NumGoroutine(),
				)
				evictsNumPer5Sec = 0
				evictsMemPer5Sec = 0
			}
		}
	}()
}
