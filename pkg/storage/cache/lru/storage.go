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
	evictionStatCh = make(chan evictionStat, synced.PreallocateBatchSize)
)

// evictionStat carries statistics for each eviction batch.
type evictionStat struct {
	items int     // Number of evicted items
	mem   uintptr // Total freed Weight
}

// Storage is a Weight-aware, sharded Storage cache with background eviction and refresh support.
type Storage struct {
	ctx             context.Context               // Main context for lifecycle control
	cfg             *config.Cache                 // Cache configuration
	shardedMap      *sharded.Map[*model.Response] // Sharded storage for cache entries
	refresher       Refresher                     // Background refresher (see refresher.go)
	balancer        Balancer                      // Helps pick shards to evict from
	backend         repository.Backender          // Remote backend server.
	mem             int64                         // Current Weight usage (bytes)
	memoryLimit     int64                         // Hard limit for Weight usage (bytes)
	memoryThreshold int64                         // Threshold for triggering eviction (bytes)
}

// NewStorage constructs a new Storage cache instance and launches eviction and refresh routines.
func NewStorage(
	ctx context.Context,
	cfg *config.Cache,
	balancer Balancer,
	refresher Refresher,
	backend repository.Backender,
	shardedMap *sharded.Map[*model.Response],
) *Storage {
	storage := &Storage{
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
	storage.shardedMap.WalkShards(func(shardKey uint64, shard *sharded.Shard[*model.Response]) {
		storage.balancer.Register(shard)
	})

	// Launch background refresher and evictors.
	storage.refresher.RunRefresher()

	storage.runEvictor()
	if cfg.IsDebugOn() {
		storage.runLogger()
	}

	// Load dump of it exists in public/dump dir.
	storage.loadDumpIfExists()

	return storage
}

// Get retrieves a response by request and bumps its Storage position.
// Returns: (response, releaser, found).
func (c *Storage) Get(req *model.Request) (*model.Response, *sharded.Releaser[*model.Response], bool) {
	resp, releaser, found := c.shardedMap.Get(req.Key(), req.ShardKey())
	if found {
		c.touch(resp)
		return resp, releaser, true
	}
	return nil, nil, false
}

// Set inserts or updates a response in the cache, updating Weight usage and Storage position.
func (c *Storage) Set(new *model.Response) *sharded.Releaser[*model.Response] {
	existing, releaser, found := c.shardedMap.Get(new.Request().Key(), new.Request().ShardKey())
	if found {
		c.update(existing, new)
		return releaser
	}
	return c.set(new)
}

// touch bumps the Storage position of an existing entry (MoveToFront) and increases its refcount.
func (c *Storage) touch(existing *model.Response) {
	existing.IncRefCount()
	c.balancer.Move(existing.Request().ShardKey(), existing.LruListElement())
}

// update refreshes Weight accounting and Storage position for an updated entry.
func (c *Storage) update(existing, new *model.Response) {
	atomic.AddInt64(&c.mem, int64(existing.Weight()-new.Weight()))
	c.touch(existing)
}

// set inserts a new response, updates Weight usage and registers in balancer.
func (c *Storage) set(new *model.Response) *sharded.Releaser[*model.Response] {
	atomic.AddInt64(&c.mem, int64(new.Weight()))
	releaser := c.shardedMap.Set(new)
	c.balancer.Set(new)
	return releaser
}

// runLogger emits detailed stats about evictions, Weight, and GC activity every 5 seconds if debugging is enabled.
func (c *Storage) runLogger() {
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

func (c *Storage) loadDumpIfExists() {
	if err := c.LoadFromDir(c.ctx, dumpDir); err != nil {
		log.Warn().Msg("failed to load dump: " + err.Error())
	}
}

func (c *Storage) Stop() {
	// spawn a new one with limit for k8s timeout before the service will be received SIGKILL
	dumpCtx, dumpCancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer dumpCancel()

	if err := c.DumpToDir(dumpCtx, dumpDir); err != nil {
		log.Err(err).Msg("failed to dump cache")
	}
}
