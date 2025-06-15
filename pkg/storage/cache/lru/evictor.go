package lru

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/consts"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"time"
)

const evictionsPerSample = 8

// runEvictor launches multiple evictor goroutines for concurrent eviction.
func (c *Storage) runEvictor() {
	go c.evictor()
}

// evictor is the main background eviction loop for one worker.
// Each worker tries to bring Weight usage under the threshold by evicting from most loaded shards.
func (c *Storage) evictor() {
	t := utils.NewTicker(c.ctx, time.Second)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-t:
			items, freedMem := c.evictUntilWithinLimit()
			if c.cfg.IsDebugOn() && (items > 0 || freedMem > 0) {
				evictionStatCh <- evictionStat{items: items, freedMem: freedMem}
			}
		}
	}
}

func (c *Storage) usedMem() int64 {
	used := c.shardedMap.Mem()
	used += model.DataPool.Mem()
	used += model.KeyBufferPool.Mem()
	used += model.ResponsePool.Mem()
	used += model.TagsSlicesPool.Mem()
	used += model.GzipBufferPool.Mem()
	used += synced.ResponseReaderBufferPool.Mem()
	used += model.GzipWriterPool.Mem()
	used += model.HasherPool.Mem()
	used += consts.PtrBytesWeight * c.shardedMap.Len() // balancer lru list weight
	return used
}

// shouldEvict checks if current Weight usage has reached or exceeded the threshold.
func (c *Storage) shouldEvict() bool {
	return c.usedMem() >= c.memoryThreshold
}

// evictUntilWithinLimit repeatedly removes entries from the most loaded shard (tail of Storage)
// until Weight drops below threshold or no more can be evicted.
func (c *Storage) evictUntilWithinLimit() (items int, mem int64) {
	maxShards := float64(sharded.ShardCount)
	percentage := int(maxShards * 0.25)

	shardOffset := 0
	for c.shouldEvict() {
		shardOffset++
		if shardOffset >= percentage {
			shardOffset = 0
		}

		c.balancer.Rebalance()
		shard, found := c.balancer.MostLoadedSampled(shardOffset)
		if !found {
			//log.Info().Msgf("shard not found")
			continue
		}

		lru := shard.lruList
		if lru.Len() == 0 {
			//log.Info().Msgf("break lru.Len()")
			break
		}

		//log.Info().Msgf("shard found: %d", shard.shard.ID())

		offset := 0
		evictions := 0
		for c.shouldEvict() {
			lru.Lock()
			el, ok := lru.NextUnlocked(offset)
			if !ok {
				lru.Unlock()
				//log.Info().Msgf("break !ok, len: %d", lru.Len())
				break
			}

			if el.Value().IsDoomed() {
				lru.Unlock()
				offset++
				//log.Info().Msgf("continue")
				continue
			}

			key := el.Value().Key()
			shardKey := el.Value().ShardKey()
			lru.Unlock()

			freedMem, isHit := c.del(key, shardKey)
			if isHit {
				items++
				evictions++
				mem += freedMem
			} else {
				//log.Info().Msgf("break !isHit")
			}

			offset++
		}
	}
	return
}
