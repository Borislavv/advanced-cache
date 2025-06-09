package lru

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"runtime"
	"strconv"
	"time"
	"unsafe"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
)

var (
	maxEvictors    = runtime.GOMAXPROCS(0)
	evictionStatCh = make(chan evictionStat, maxEvictors*100)
)

type evictionStat struct {
	items int
	mem   uintptr
}

type LRU struct {
	ctx             context.Context
	cfg             *config.Config
	shardedMap      *sharded.Map[*model.Response]
	balancer        *Balancer
	memoryLimit     uintptr
	memoryThreshold uintptr
}

func NewLRU(ctx context.Context, cfg *config.Config, shardedMap *sharded.Map[*model.Response]) *LRU {
	const maxMemoryCoefficient uintptr = 98

	lru := &LRU{
		ctx:             ctx,
		cfg:             cfg,
		shardedMap:      shardedMap,
		memoryThreshold: uintptr(float64(cfg.MemoryLimit) * cfg.MemoryFillThreshold),
		memoryLimit:     (uintptr(cfg.MemoryLimit) / 100) * maxMemoryCoefficient,
	}

	lru.balancer = NewBalancer(lru.shardedMap)
	lru.shardedMap.WalkShards(func(shardKey uint64, shard *sharded.Shard[*model.Response]) {
		lru.balancer.register(shard)
	})

	lru.runEvictors()
	if cfg.IsDebugOn() {
		lru.runLogDebugInfo()
	}

	return lru
}

func (c *LRU) Get(req *model.Request) (*model.Response, *sharded.Releaser[*model.Response], bool) {
	resp, releaser, found := c.shardedMap.Get(req.Key(), req.ShardKey())
	if found {
		c.touch(resp)
		return resp, releaser, true
	}
	return nil, nil, false
}

func (c *LRU) Set(new *model.Response) *sharded.Releaser[*model.Response] {
	existing, releaser, found := c.shardedMap.Get(new.GetRequest().Key(), new.GetRequest().ShardKey())
	if found {
		c.update(existing, new)
		return releaser
	}
	c.set(new)
	return nil
}

func (c *LRU) touch(existing *model.Response) {
	existing.IncRefCount()
	c.balancer.move(existing.GetRequest().ShardKey(), existing.GetListElement())
	log.Info().Msg("found")
}

func (c *LRU) update(existing, new *model.Response) {
	c.touch(existing)
	c.balancer.move(new.GetRequest().ShardKey(), existing.GetListElement())
}

func (c *LRU) set(new *model.Response) {
	c.balancer.set(new)
	c.shardedMap.Set(new.GetRequest().Key(), new)
}

func (c *LRU) del(req *model.Request) (freedMem uintptr, isHit bool) {
	return c.balancer.remove(req.Key(), req.ShardKey())
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
			time.Sleep(time.Millisecond * 250)
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
		freedMem, isHit := c.del(back.Value)
		if !isHit {
			continue
		}
		items++
		mem += freedMem
	}
	return
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
