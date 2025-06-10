package lru

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/wheel"
	wheelmodel "github.com/Borislavv/traefik-http-cache-plugin/pkg/wheel/model"
	"github.com/rs/zerolog/log"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	maxEvictors    = 4
	evictionStatCh = make(chan evictionStat, maxEvictors*synced.PreallocationBatchSize)
)

type evictionStat struct {
	items int
	mem   uintptr
}

type LRU struct {
	ctx             context.Context
	cfg             *config.Config
	shardedMap      *sharded.Map[*model.Response]
	timeWheel       *wheel.OfTime[wheelmodel.Spoke]
	balancer        *Balancer
	mem             int64
	memoryLimit     int64
	memoryThreshold int64
}

func NewLRU(ctx context.Context, cfg *config.Config, timeWheel *wheel.OfTime[wheelmodel.Spoke], shardedMap *sharded.Map[*model.Response]) *LRU {
	lru := &LRU{
		ctx:             ctx,
		cfg:             cfg,
		timeWheel:       timeWheel,
		shardedMap:      shardedMap,
		memoryThreshold: int64(float64(cfg.MemoryLimit) * cfg.MemoryFillThreshold),
		memoryLimit:     int64(cfg.MemoryLimit)/100 - 1,
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

func (c *LRU) GetBy(key uint64, shard uint64) (resp *model.Response, releaser *sharded.Releaser[*model.Response], isHit bool) {
	resp, releaser, found := c.shardedMap.Get(key, shard)
	if found {
		c.touch(resp)
		return resp, releaser, true
	}
	return nil, nil, false
}

func (c *LRU) Set(new *model.Response) *sharded.Releaser[*model.Response] {
	existing, releaser, found := c.shardedMap.Get(new.Request().Key(), new.Request().ShardKey())
	if found {
		c.update(existing, new)
		return releaser
	}
	c.set(new)
	return nil
}

func (c *LRU) touch(existing *model.Response) {
	existing.IncRefCount()
	c.balancer.move(existing.Request().ShardKey(), existing.LruListElement())
}

func (c *LRU) update(existing, new *model.Response) {
	atomic.AddInt64(&c.mem, int64(existing.Size()-new.Size()))
	c.touch(existing)
	c.timeWheel.Touch(existing)
	c.balancer.move(new.Request().ShardKey(), existing.LruListElement())
}

func (c *LRU) set(new *model.Response) {
	atomic.AddInt64(&c.mem, int64(new.Size()))
	c.balancer.set(new)
	c.timeWheel.Add(new)
	c.shardedMap.Set(new)
}

func (c *LRU) del(req *model.Request) (freedMem uintptr, isHit bool) {
	return c.balancer.remove(req.Key(), req.ShardKey())
}

func (c *LRU) runEvictors() {
	for id := 0; id < maxEvictors; id++ {
		go c.evictor(id)
	}
}

func (c *LRU) evictor(id int) {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if c.shouldEvict() {
				items, memory := c.evictUntilWithinLimit(id)
				if c.cfg.IsDebugOn() && (items > 0 || memory > 0) {
					select {
					case evictionStatCh <- evictionStat{items: items, mem: memory}:
					default:
					}
				}
			}
			time.Sleep(time.Second)
		}
	}
}

func (c *LRU) shouldEvict() bool {
	return atomic.LoadInt64(&c.mem) >= c.memoryThreshold
}

func (c *LRU) evictUntilWithinLimit(id int) (items int, mem uintptr) {
	for atomic.LoadInt64(&c.mem) > c.memoryThreshold {
		shard, found := c.balancer.mostLoaded(id)
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
		atomic.AddInt64(&c.mem, -int64(freedMem))
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

				var (
					mem        = strconv.Itoa(int(atomic.LoadInt64(&c.mem)))
					length     = strconv.Itoa(int(c.shardedMap.Len()))
					gc         = strconv.Itoa(int(m.NumGC))
					limit      = strconv.Itoa(int(c.cfg.MemoryLimit))
					goroutines = strconv.Itoa(runtime.NumGoroutine())
					alloc      = utils.FmtMem(uintptr(m.Alloc))
				)

				log.
					Info().
					Str("target", "lru").
					Str("mem", mem).
					Str("len", length).
					Str("GC", gc).
					Str("memLimit", limit).
					Str("goroutines", goroutines).
					Msgf(
						"[lru][5s] evicted (items: %d, mem: %dB), "+
							"storage (usage: %sB, len: %s, limit: %sB), sys (alloc: %s, goroutines: %s, GC: %s)",
						evictsNumPer5Sec, evictsMemPer5Sec, mem, length, limit, alloc, goroutines, gc,
					)

				evictsNumPer5Sec = 0
				evictsMemPer5Sec = 0
			}
		}
	}()
}
