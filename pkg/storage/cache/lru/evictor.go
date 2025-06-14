package lru

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	"sync/atomic"
	"time"
)

// runEvictor launches multiple evictor goroutines for concurrent eviction.
func (c *Storage) runEvictor() {
	go c.evictor()
}

// evictor is the main background eviction loop for one worker.
// Each worker tries to bring Weight usage under the threshold by evicting from most loaded shards.
func (c *Storage) evictor() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if c.shouldEvict() {
				items, memory := c.evictUntilWithinLimit()
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

// shouldEvict checks if current Weight usage has reached or exceeded the threshold.
func (c *Storage) shouldEvict() bool {
	return atomic.LoadInt64(&c.mem) >= c.memoryThreshold
}

// evictUntilWithinLimit repeatedly removes entries from the most loaded shard (tail of Storage)
// until Weight drops below threshold or no more can be evicted.
func (c *Storage) evictUntilWithinLimit() (items int, mem uintptr) {
	for atomic.LoadInt64(&c.mem) > c.memoryThreshold {
		shard, found := c.balancer.MostLoaded()
		if !found {
			break
		}

		shard.lruList.Walk(list.FromBack, func(l *list.List[*model.Response], el *list.Element[*model.Response]) bool {
			old := el.Value().RefCount()
			if el.Value().CASRefCount(old, old+1) && !el.Value().IsDoomed() {
				select {
				case <-c.ctx.Done():
					return false
				default:
					el.Value().DecRefCount()
					freedMem, isHit := c.balancer.Remove(el.Value().Key(), el.Value().ShardKey())
					if !isHit {
						return true
					}
					items++
					mem += freedMem
					atomic.AddInt64(&c.mem, -int64(freedMem))
					return true
				}
			}
			return true
		})
	}
	return
}
