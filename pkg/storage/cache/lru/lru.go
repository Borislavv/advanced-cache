package lru

import (
	"context"
	"runtime"
	"time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	times "github.com/Borislavv/traefik-http-cache-plugin/pkg/time"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
)

var (
	maxEvictors    = runtime.GOMAXPROCS(0)
	evictionStatCh = make(chan evictionStat, maxEvictors*4)
)

type evictionStat struct {
	items int
	mem   uintptr
}

type LRU struct {
	ctx               context.Context
	cfg               *config.Config
	shardedMap        *sharded.Map[*model.Response]
	timeWheel         *times.Wheel
	balancer          *Balancer
	evictionThreshold uintptr
	evictionTriggerCh chan struct{}
}

func NewLRU(ctx context.Context, cfg *config.Config, timeWheel *times.Wheel, shardedMap *sharded.Map[*model.Response]) *LRU {
	lru := &LRU{
		ctx:               ctx,
		cfg:               cfg,
		timeWheel:         timeWheel,
		shardedMap:        shardedMap,
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

func (c *LRU) Get(req *model.Request) (*model.Response, bool) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	resp, found := c.shardedMap.Get(key, shardKey)
	if found {
		c.onFound(shardKey, resp, nil)
		return resp, true
	}
	return nil, false
}

func (c *LRU) onFound(shardKey uint64, target *model.Response, source *model.Response) {
	if source != nil {
		target.SetData(source.GetData())
	}
	c.balancer.move(shardKey, target.GetListElement())
}

func (c *LRU) Set(newResp *model.Response) {
	key := newResp.GetRequest().UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	shard := c.shardedMap.Shard(shardKey)

	resp, found := shard.Get(key)
	if found {
		c.onFound(shardKey, resp, newResp)
		return
	}

	c.onSet(key, shard, newResp)
}

func (c *LRU) onSet(key uint64, shard *sharded.Shard[*model.Response], resp *model.Response) {
	resp.SetShardKey(shard.ID())

	if c.shouldEvict(resp.Size()) {
		c.triggerEviction()
	}

	c.balancer.set(resp)
	shard.Set(key, resp)
	c.timeWheel.Add(key, shard.ID(), c.cfg.RevalidateInterval)
}

func (c *LRU) del(req *model.Request) (*model.Response, bool) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	return c.balancer.remove(key, shardKey)
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
				items, memory := c.evictUntilWithinLimit()
				if c.cfg.IsDebugOn() && (items > 0 || memory > 0) {
					evictionStatCh <- evictionStat{items: items, mem: memory}
				}
			}
		}
	}()
}

func (c *LRU) shouldEvict(respSize uintptr) bool {
	return c.shardedMap.Mem()+respSize > c.evictionThreshold
}

func (c *LRU) triggerEviction() {
	select {
	case c.evictionTriggerCh <- struct{}{}:
	default:
		// if async evictor cannot process your request
		//  otherwise use a manual approach
		items, memory := c.evictUntilWithinLimit()
		if c.cfg.IsDebugOn() && (items > 0 || memory > 0) {
			evictionStatCh <- evictionStat{items: items, mem: memory}
		}
	}
}

func (c *LRU) evictUntilWithinLimit() (items int, mem uintptr) {
	var (
		limit            = c.evictionThreshold
		avgItemsPerShard = float64(c.shardedMap.Len()) / float64(sharded.ShardCount)
		preferMostLoaded = avgItemsPerShard > 4.0
	)

	const maxEvictIterations = 2048
	for i := 0; i < maxEvictIterations && c.shardedMap.Mem() > limit; i++ {
		var (
			found bool
			shard *shardNode
		)

		if preferMostLoaded {
			shard, found = c.balancer.mostLoaded()
		} else {
			shard, found = c.balancer.anyLoaded()
		}
		if !found {
			break
		}

		back := shard.lruList.Back()
		if back == nil {
			c.balancer.rebalance(shard)
			continue
		}

		if resp, ok := c.del(back.Value); ok {
			items++
			mem += resp.Size()
		} else {
			shard.lruList.Remove(back)
		}
	}

	return
}

func (c *LRU) runLogDebugInfo() {
	go func() {
		ticker := utils.NewTicker(c.ctx, 5*time.Second)
		var (
			evictsNumPer5Sec int
			evictsMemPer5Sec uintptr
		)
		for {
			select {
			case <-c.ctx.Done():
				return
			case stat := <-evictionStatCh:
				evictsNumPer5Sec += stat.items
				evictsMemPer5Sec += stat.mem
			case <-ticker:
				log.Debug().Msgf("[lru]: evicted [n: %d, mem: %d bytes] (5s), memory usage: %s (limit: %s), storage len: %d, goroutines: %d.",
					evictsNumPer5Sec, evictsMemPer5Sec,
					utils.FmtMem(c.shardedMap.Mem()), utils.FmtMem(uintptr(c.cfg.MemoryLimit)),
					c.shardedMap.Len(), runtime.NumGoroutine(),
				)
				evictsNumPer5Sec = 0
				evictsMemPer5Sec = 0
			}
		}
	}()
}
