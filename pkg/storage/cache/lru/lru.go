package lru

import (
	"context"
	"runtime"
	"strconv"
	"time"
	"unsafe"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	times "github.com/Borislavv/traefik-http-cache-plugin/pkg/time"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
)

var (
	maxEvictors    = runtime.GOMAXPROCS(0)
	evictionStatCh = make(chan evictionStat, maxEvictors*100)
	trashList      = list.New[*model.Response](true)
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

	lru.runEvictors()
	lru.runTrashCollector()

	if cfg.IsDebugOn() {
		lru.runLogDebugInfo()
	}

	return lru
}

func (c *LRU) Get(req *model.Request) (*model.Response, func(), bool) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	resp, found := c.shardedMap.Get(key, shardKey)
	if found {
		resp.IncRefCount()
		c.balancer.move(shardKey, resp.GetListElement())
		return resp, func() { resp.DcrRefCount() }, true
	}
	return nil, func() {}, false
}

func (c *LRU) Set(resp *model.Response) {
	key := resp.GetRequest().UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	shard := c.shardedMap.Shard(shardKey)

	existing, found := c.shardedMap.Get(key, shardKey)
	if found {
		existing.IncRefCount()
		defer existing.DcrRefCount()
		existing.SetData(resp.GetData())
		c.balancer.move(shardKey, existing.GetListElement())
		return
	}

	resp.SetShardKey(shard.ID())
	el := c.balancer.set(resp)
	resp.SetListElement(el)
	shard.Set(key, resp)
	c.timeWheel.Add(key, shard.ID(), c.cfg.RevalidateInterval)
}

func (c *LRU) del(req *model.Request) (*model.Response, bool) {
	key := req.UniqueKey()
	shardKey := c.shardedMap.GetShardKey(key)
	return c.balancer.remove(key, shardKey)
}

func (c *LRU) runEvictors() {
	for i := 0; i < maxEvictors; i++ {
		go c.evictor()
	}
}

func (c *LRU) evictor() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if c.shouldEvict() {
				items, memory := c.evictUntilWithinLimit()
				if c.cfg.IsDebugOn() && (items > 0 || memory > 0) {
					select {
					case evictionStatCh <- evictionStat{items: items, mem: memory}:
					default:
					}
				}
			}
			runtime.Gosched()
			time.Sleep(time.Millisecond * 100)
		}
	}
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
	for c.memory() > c.memoryThreshold {
		shard, found := c.balancer.mostLoaded()
		if !found {
			break
		}
		back := shard.lruList.Back()
		if back == nil || back.Value == nil {
			shard.lruList.Remove(back)
			continue
		}
		resp, isHit := c.del(back.Value)
		if !isHit {
			continue
		}
		items++
		mem += resp.Size()
		for i := 0; i < 10; i++ {
			if resp.CASRefCount(0, -1) {
				resp.Free()
				continue
			}
		}
		markAsDoomed(resp)
	}
	return
}

func markAsDoomed(resp *model.Response) {
	if !resp.IsFreed() && resp.RefCount() > -1 {
		trashList.PushBack(resp)
	}
}

func (c *LRU) runTrashCollector() {
	go func() {
		ticker := time.NewTicker(time.Millisecond * 250)
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				var freedItems int
				var freedMem uintptr
				for el := trashList.Front(); el != nil; {
					next := el.Next()
					resp := el.Value

					for i := 0; i < 10; i++ {
						if resp.CASRefCount(0, -1) {
							trashList.Remove(el)
							resp.Free()

							freedMem += resp.Size()
							freedItems++

							if c.cfg.IsDebugOn() && freedItems > 0 {
								log.Debug().Msgf("[trash]: freed %d doomed responses (%d bytes)", freedItems, freedMem)
							}

							continue
						}
					}

					el = next
				}
			}
		}
	}()
}

func (c *LRU) runLogDebugInfo() {
	go func() {
		ticker := utils.NewTicker(c.ctx, 5*time.Second)
		var evictsNumPer5Sec int
		var evictsMemPer5Sec uintptr
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
					"storage [memUsage: %s, memLimit: %s, len: %d], sys [alloc: %s, totalAlloc: %s, sysAlloc: %s, routines: %d, GC: %s]",
					evictsNumPer5Sec, evictsMemPer5Sec,
					utils.FmtMem(c.memory()), utils.FmtMem(uintptr(c.cfg.MemoryLimit)),
					c.shardedMap.Len(), utils.FmtMem(uintptr(m.Alloc)), utils.FmtMem(uintptr(m.TotalAlloc)), utils.FmtMem(uintptr(m.Sys)),
					runtime.NumGoroutine(), strconv.Itoa(int(m.NumGC)))
				evictsNumPer5Sec = 0
				evictsMemPer5Sec = 0
			}
		}
	}()
}
