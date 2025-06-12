package lru

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

const dumpDir = "public/dump"

var (
	maxEvictors    = 4 // Number of concurrent evictor goroutines
	evictionStatCh = make(chan evictionStat, maxEvictors*synced.PreallocationBatchSize)
)

// evictionStat carries statistics for each eviction batch.
type evictionStat struct {
	items int     // Number of evicted items
	mem   uintptr // Total freed Memory
}

// LRU is a Memory-aware, sharded LRU cache with background eviction and refresh support.
type LRU struct {
	ctx             context.Context               // Main context for lifecycle control
	cfg             *config.Config                // Cache configuration
	shardedMap      *sharded.Map[*model.Response] // Sharded storage for cache entries
	refresher       Refresher                     // Background refresher (see refresher.go)
	balancer        Balancer                      // Helps pick shards to evict from
	backend         repository.Backender          // Remote backend server.
	mem             int64                         // Current Memory usage (bytes)
	memoryLimit     int64                         // Hard limit for Memory usage (bytes)
	memoryThreshold int64                         // Threshold for triggering eviction (bytes)
}

// NewLRU constructs a new LRU cache instance and launches eviction and refresh routines.
func NewLRU(
	ctx context.Context,
	cfg *config.Config,
	balancer Balancer,
	refresher Refresher,
	backend repository.Backender,
	shardedMap *sharded.Map[*model.Response],
) *LRU {
	lru := &LRU{
		ctx:             ctx,
		cfg:             cfg,
		shardedMap:      shardedMap,
		refresher:       refresher,
		balancer:        balancer,
		backend:         backend,
		memoryThreshold: int64(float64(cfg.MemoryLimit) * cfg.MemoryFillThreshold),
		memoryLimit:     int64(cfg.MemoryLimit)/100 - 8, // Defensive: avoid going exactly to limit
	}

	// Register all existing shards with the balancer.
	lru.shardedMap.WalkShards(func(shardKey uint64, shard *sharded.Shard[*model.Response]) {
		lru.balancer.Register(shard)
	})

	// Launch background refresher and evictors.
	lru.refresher.RunRefresher()
	lru.runEvictors()
	if cfg.IsDebugOn() {
		lru.runLogger()
	}

	// Load dump of it exists in public/dump dir.
	lru.loadDumpIfExists()

	return lru
}

// Get retrieves a response by request and bumps its LRU position.
// Returns: (response, releaser, found).
func (c *LRU) Get(req *model.Request) (*model.Response, *sharded.Releaser[*model.Response], bool) {
	resp, releaser, found := c.shardedMap.Get(req.Key(), req.ShardKey())
	if found {
		c.touch(resp)
		return resp, releaser, true
	}
	return nil, nil, false
}

// Set inserts or updates a response in the cache, updating Memory usage and LRU position.
func (c *LRU) Set(new *model.Response) *sharded.Releaser[*model.Response] {
	existing, releaser, found := c.shardedMap.Get(new.Request().Key(), new.Request().ShardKey())
	if found {
		c.update(existing, new)
		return releaser
	}
	return c.set(new)
}

// touch bumps the LRU position of an existing entry (MoveToFront) and increases its refcount.
func (c *LRU) touch(existing *model.Response) {
	existing.IncRefCount()
	c.balancer.Move(existing.Request().ShardKey(), existing.LruListElement())
}

// update refreshes Memory accounting and LRU position for an updated entry.
func (c *LRU) update(existing, new *model.Response) {
	atomic.AddInt64(&c.mem, int64(existing.Size()-new.Size()))
	c.touch(existing)
	c.balancer.Move(new.Request().ShardKey(), existing.LruListElement())
}

// set inserts a new response, updates Memory usage and registers in balancer.
func (c *LRU) set(new *model.Response) *sharded.Releaser[*model.Response] {
	atomic.AddInt64(&c.mem, int64(new.Size()))
	c.balancer.Set(new)
	return c.shardedMap.Set(new)
}

// del removes an entry and returns the amount of freed Memory and whether it was present.
func (c *LRU) del(resp *model.Response) (freedMem uintptr, isHit bool) {
	return c.balancer.Remove(resp.Key(), resp.ShardKey())
}

// runEvictors launches multiple evictor goroutines for concurrent eviction.
func (c *LRU) runEvictors() {
	for id := 0; id < maxEvictors; id++ {
		go c.evictor(id)
	}
}

// evictor is the main background eviction loop for one worker.
// Each worker tries to bring Memory usage under the threshold by evicting from most loaded shards.
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
					case <-c.ctx.Done():
						return
					case evictionStatCh <- evictionStat{items: items, mem: memory}:
					}
				}
			}
			time.Sleep(time.Second)
		}
	}
}

// shouldEvict checks if current Memory usage has reached or exceeded the threshold.
func (c *LRU) shouldEvict() bool {
	return atomic.LoadInt64(&c.mem) >= c.memoryThreshold
}

// evictUntilWithinLimit repeatedly removes entries from the most loaded shard (tail of LRU)
// until Memory drops below threshold or no more can be evicted.
func (c *LRU) evictUntilWithinLimit(id int) (items int, mem uintptr) {
	for atomic.LoadInt64(&c.mem) > c.memoryThreshold {
		shard, found := c.balancer.MostLoaded(id)
		if !found {
			break
		}
		back := shard.lruList.Back()
		if back == nil || back.Value == nil {
			// Defensive: try to Remove empty element to avoid leaks
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

// runLogger emits detailed stats about evictions, Memory, and GC activity every 5 seconds if debugging is enabled.
func (c *LRU) runLogger() {
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
					mem        = utils.FmtMem(uintptr(atomic.LoadInt64(&c.mem)))
					length     = strconv.Itoa(int(c.shardedMap.Len()))
					gc         = strconv.Itoa(int(m.NumGC))
					limit      = utils.FmtMem(uintptr(c.cfg.MemoryLimit))
					goroutines = strconv.Itoa(runtime.NumGoroutine())
					alloc      = utils.FmtMem(uintptr(m.Alloc))
					freedMem   = utils.FmtMem(evictsMemPer5Sec)
				)

				log.
					Info().
					//Str("target", "lru").
					//Str("mem", mem).
					//Str("len", length).
					//Str("GC", gc).
					//Str("memLimit", limit).
					//Str("goroutines", goroutines).
					Msgf(
						"[lru][5s] evicted (items: %d, mem: %s), "+
							"storage (usage: %s, len: %s, limit: %s), sys (alloc: %s, goroutines: %s, GC: %s)",
						evictsNumPer5Sec, freedMem, mem, length, limit, alloc, goroutines, gc,
					)

				evictsNumPer5Sec = 0
				evictsMemPer5Sec = 0
			}
		}
	}()
}

func (c *LRU) loadDumpIfExists() {
	if err := c.LoadFromDir(c.ctx, dumpDir); err != nil {
		log.Warn().Msg("failed to load dump: " + err.Error())
	}
}

func (c *LRU) Stop() {
	// spawn a new one with limit for k8s timeout before the service will be received SIGKILL
	dumpCtx, dumpCancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer dumpCancel()

	if err := c.DumpToDir(dumpCtx, dumpDir); err != nil {
		log.Err(err).Msg("failed to dump cache")
	}
}
