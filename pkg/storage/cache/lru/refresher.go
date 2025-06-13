package lru

import (
	"context"
	"fmt"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
	"math/rand"
	"strconv"
	"time"
)

const (
	rateLimit      = 16 // Global limiter: maximum concurrent refreshes across all shards
	rateLimitBurst = 8  // Global limiter: maximum parallel requests.
	refreshSamples = 16 // Number of items to sample per shard per refresh tick
	// Max refreshes per second = rateLimit(32) * refreshSamples(32) = 1024.
)

var (
	refreshSuccessNumCh = make(chan struct{}, synced.PreallocateBatchSize) // Successful refreshes counter channel
	refreshErroredNumCh = make(chan struct{}, synced.PreallocateBatchSize) // Failed refreshes counter channel
)

type Refresher interface {
	RunRefresher()
}

// Refresh is responsible for background refreshing of cache entries.
// It periodically samples random shards and randomly selects "cold" entries
// (from the end of each shard's Storage list) to refresh if necessary.
type Refresh struct {
	ctx               context.Context
	cfg               *config.Cache
	balancer          *Balance
	shardsSamplesCh   chan *ShardNode
	refreshRespCh     chan *model.Response
	shardRateLimiter  *rate.Limiter
	updateRateLimiter *rate.Limiter
}

// NewRefresher constructs a Refresh.
func NewRefresher(ctx context.Context, cfg *config.Cache, balancer *Balance) *Refresh {
	return &Refresh{
		ctx:               ctx,
		cfg:               cfg,
		balancer:          balancer,
		shardsSamplesCh:   make(chan *ShardNode),
		refreshRespCh:     make(chan *model.Response, rateLimit),
		shardRateLimiter:  rate.NewLimiter(rateLimit, rateLimitBurst),
		updateRateLimiter: rate.NewLimiter(rateLimit, rateLimitBurst),
	}
}

// RunRefresher starts the refresher background loop.
// It runs a logger (if debugging is enabled), spawns a provider for sampling shards,
// and continuously processes shard samples for candidate responses to refresh.
func (r *Refresh) RunRefresher() {
	go func() {
		if r.cfg.IsDebugOn() {
			r.runLogger()
		}

		for { // Throttling (max 512 checks per second)
			if err := r.shardRateLimiter.Wait(r.ctx); err != nil {
				return
			}

			r.refreshNode(r.balancer.RandShardNode())
		}
	}()
}

// refresh attempts to refresh the given response via Revalidate.
// If successful, increments the refresh metric (in debug mode); otherwise increments the error metric.
func (r *Refresh) refresh(resp *model.Response) {
	defer resp.DecRefCount()

	if err := resp.Revalidate(r.ctx); err != nil {
		log.
			Err(err).
			Str("key", fmt.Sprintf("%v", resp.Key())).
			Str("shardKey", fmt.Sprintf("%v", resp.ShardKey())).
			Str("query", string(resp.ToQuery())).
			Msg("response refresh failed")

		if r.cfg.IsDebugOn() {
			select {
			case <-r.ctx.Done():
			case refreshErroredNumCh <- struct{}{}:
			}
		}
		return
	}

	if r.cfg.IsDebugOn() {
		select {
		case <-r.ctx.Done():
		case refreshSuccessNumCh <- struct{}{}:
		}
	}
}

// refreshNode selects up to refreshSamples entries from the end of the given shard's Storage list.
// For each candidate, if ShouldBeRefreshed() returns true and shardRateLimiter limiting allows, triggers an asynchronous refresh.
func (r *Refresh) refreshNode(node *ShardNode) {
	lru := node.lruList
	length := lru.Len()
	if length == 0 {
		return
	}

	// Uses round-robbin algo. when storage len is too short for sampling.
	if length <= refreshSamples*10 {
		lru.Walk(list.FromBack, func(l *list.List[*model.Response], el *list.Element[*model.Response]) bool {
			if el.Value().IsDoomed() {
				return true
			}

			el.Value().IncRefCount()
			defer el.Value().DecRefCount()

			select {
			case <-r.ctx.Done():
				return false
			default:
				if el.Value().ShouldBeRefreshed() {
					go r.refresh(el.Value())
				}
				return true
			}
		})
		return
	}

	// Uses sampling when storage len reached the threshold (refreshSamples*100=~320).
	// Sampling algo. will be more effective from the threshold.
	for i := 0; i < refreshSamples; i++ {
		skipped := 0
		lru.Walk(list.FromBack, func(l *list.List[*model.Response], el *list.Element[*model.Response]) bool {
			if el.Value().IsDoomed() {
				return true
			}

			el.Value().IncRefCount()
			defer el.Value().DecRefCount()

			select {
			case <-r.ctx.Done():
				return false
			default:
				offset := rand.Intn(length)
				if skipped < offset {
					skipped++
					return true
				}
				if el.Value().ShouldBeRefreshed() {
					go r.refresh(el.Value())
				}
				return true
			}
		})
	}
}

// runLogger periodically logs the number of successful and failed refresh attempts.
// This runs only if debugging is enabled in the config.
func (r *Refresh) runLogger() {
	go func() {
		refreshesNumPer5Sec := 0
		erroredNumPer5Sec := 0
		ticker := utils.NewTicker(r.ctx, 5*time.Second)
		for {
			select {
			case <-r.ctx.Done():
				return
			case <-refreshSuccessNumCh:
				refreshesNumPer5Sec++
			case <-refreshErroredNumCh:
				erroredNumPer5Sec++
			case <-ticker:
				var (
					errorsNum  = strconv.Itoa(erroredNumPer5Sec)
					successNum = strconv.Itoa(refreshesNumPer5Sec)
				)

				log.
					Info().
					//Str("target", "refresher").
					//Str("processed", successNum).
					//Str("errored", errorsNum).
					Msgf("[refresher][5s] success %s, errors: %s", successNum, errorsNum)

				refreshesNumPer5Sec = 0
				erroredNumPer5Sec = 0
			}
		}
	}()
}
