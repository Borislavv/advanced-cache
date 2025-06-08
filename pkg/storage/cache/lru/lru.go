package lru

import (
	"context"
	"runtime"
	"strconv"
	"time"
	"unsafe"

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
	ctx             context.Context
	cfg             *config.Config
	shardedMap      *sharded.Map[*model.Response]
	timeWheel       *times.Wheel
	balancer        *Balancer
	memoryLimit     uintptr
	memoryThreshold uintptr
}

func NewLRU(ctx context.Context, cfg *config.Config, timeWheel *times.Wheel, shardedMap *sharded.Map[*model.Response]) *LRU {
	const maxMemoryCoefficient uintptr = 98

	lru := &LRU{
		ctx:             ctx,
		cfg:             cfg,
		timeWheel:       timeWheel,
		shardedMap:      shardedMap,
		memoryThreshold: uintptr(float64(cfg.MemoryLimit) * cfg.MemoryFillThreshold),
		memoryLimit:     (uintptr(cfg.MemoryLimit) / 100) * maxMemoryCoefficient,
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

func (c *LRU) get(key uint64, shard uint64) (*model.Response, model.ResponseAcquireFn, bool) {
	resp, found := c.shardedMap.Get(key, shard)
	if found {
		resp.IncRefCount()
		return resp, func() { resp.DcrRefCount() }, true
	}
	return nil, func() {}, false
}

func (c *LRU) Get(req *model.Request) (*model.Response, model.ResponseAcquireFn, bool) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	resp, acquire, found := c.get(key, shardKey)
	if found {
		c.onFound(shardKey, resp, nil)
		return resp, acquire, true
	}
	return nil, acquire, false
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

	resp, acquire, found := c.get(key, shardKey)
	if found {
		defer acquire()
		c.onFound(shardKey, resp, newResp)
		return
	}

	c.onSet(key, shard, newResp)
}

func (c *LRU) onSet(key uint64, shard *sharded.Shard[*model.Response], resp *model.Response) {
	resp.SetShardKey(shard.ID())
	c.balancer.set(resp)
	shard.Set(key, resp)
	c.timeWheel.Add(key, shard.ID(), c.cfg.RevalidateInterval)
}

func (c *LRU) del(req *model.Request) (resp *model.Response, isHit bool) {
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
			default:
				if c.shouldEvict() {
					items, memory := c.evictUntilWithinLimit()
					if c.cfg.IsDebugOn() && (items > 0 || memory > 0) {
						evictionStatCh <- evictionStat{items: items, mem: memory}
					}
				}
				runtime.Gosched()
				time.Sleep(time.Millisecond * 25)
			}
		}
	}()
}

func (c *LRU) shouldEvict() bool {
	return c.memory() > c.memoryThreshold
}

func (c *LRU) memory() uintptr {
	mem := unsafe.Sizeof(c)
	mem += c.shardedMap.Mem()
	mem += c.balancer.memory()
	mem += c.timeWheel.Memory()
	return mem
}

func (c *LRU) evictUntilWithinLimit() (items int, mem uintptr) {
	const maxEvictIterations = 2048
	var limit = c.memoryThreshold

loop:
	for i := 0; i < maxEvictIterations && c.memory() > limit; i++ {
		var (
			found bool
			shard *shardNode
		)

		shard, found = c.balancer.mostLoaded()
		if !found {
			break
		}

		back := shard.lruList.Back()
		if back == nil {
			c.balancer.rebalance(shard)
			continue
		}

		if back.Value == nil {
			shard.lruList.Remove(back)
			continue
		}

		resp, isHit := c.del(back.Value)
		if !isHit {
			continue
		}

		items++
		mem += resp.Size()

		/**
		 * очень мендленно и вызывает проблемы с производительностью
		 */
		trashLoopIters := 0
		for resp.RefCount() > 0 {
			if trashLoopIters >= 1000 {
				log.Warn().Msgf("attempt to free response: trash loop iters. "+
					"reached limit: 1000, refCount is still not zero: %d, is freed: %d (response leaked)", resp.RefCount(), resp.IsFreed())
				continue loop
			}
			trashLoopIters++
		}
		if resp.IsFreed() == 0 {
			resp.Free()
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
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				log.Debug().Msgf("[lru]: evicted [n: %d, mem: %d bytes] (5s), "+
					"storage [memUsage: %s, memLimit: %s, len: %d], sys [alloc: %s, totalAlloc: %s, sysAlloc: %s, routines: %d, GC: %s].",
					evictsNumPer5Sec, evictsMemPer5Sec,
					utils.FmtMem(c.memory()), utils.FmtMem(uintptr(c.cfg.MemoryLimit)),
					c.shardedMap.Len(), utils.FmtMem(uintptr(m.Alloc)), utils.FmtMem(uintptr(m.TotalAlloc)), utils.FmtMem(uintptr(m.Sys)),
					runtime.NumGoroutine(), strconv.Itoa(int(m.NumGC)),
				)
				evictsNumPer5Sec = 0
				evictsMemPer5Sec = 0
			}
		}
	}()
}
