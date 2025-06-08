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
	workersNum     = runtime.GOMAXPROCS(0)
	evictionStatCh = make(chan evictionStat, workersNum*4)
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
	lru.runTrashCollectors()

	if cfg.IsDebugOn() {
		lru.runLogDebugInfo()
	}

	return lru
}

func (c *LRU) Get(req *model.Request) (resp *model.Response, acquire func(), isHit bool) {
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
	for seed := 0; seed < workersNum; seed++ {
		go c.evictor(seed)
	}
}

func (c *LRU) evictor(seed int) {
	ticker := time.NewTicker(time.Millisecond * (15 + time.Duration(seed)))
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if c.shouldEvict() {
				items, memory := c.evictUntilWithinLimit()
				if c.cfg.IsDebugOn() && (items > 0 || memory > 0) {
					evictionStatCh <- evictionStat{items: items, mem: memory}
				}
			}
		default:
			runtime.Gosched()
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
loop:
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
		if resp.RefCount() > 0 {
			if c.cfg.IsDebugOn() {
				log.Debug().Uint64("key", resp.GetRequest().UniqueKey()).Msg("marking response as doomed")
			}
			c.markAsDoomed(resp)
			continue loop
		}
		resp.Free()
	}
	return
}

func (c *LRU) runTrashCollectors() {
	for seed := 0; seed <= runtime.GOMAXPROCS(0); seed++ {
		go c.trashCollector(seed)
	}
}

func (c *LRU) trashCollector(seed int) {
	ticker := time.NewTicker(time.Millisecond * (17 + time.Duration(seed)))
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
				if resp.RefCount() == 0 {
					freedMem += resp.Size()
					resp.Free()
					trashList.Remove(el)
					freedItems++
				}
				el = next
			}
			if c.cfg.IsDebugOn() && freedItems > 0 {
				log.Debug().Msgf("[trash]: freed %d doomed responses (%d bytes)", freedItems, freedMem)
			}
		default:
			runtime.Gosched()
		}
	}
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

func (c *LRU) markAsDoomed(resp *model.Response) {
	if !resp.IsFreed() {
		trashList.PushBack(resp)
		if resp.RefCount() == 0 {
			resp.Free()
		}
	}
}
