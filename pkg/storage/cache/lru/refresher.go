package lru

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/rate"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"math/rand"
	"strconv"
	"time"
)

const (
	rateLimit      = 512 // Global limiter: maximum concurrent refreshes across all shards
	refreshSamples = 32  // Number of items to sample per shard per refresh tick
)

var (
	refreshSuccessNumCh = make(chan struct{}, synced.PreallocationBatchSize) // Successful refreshes counter channel
	refreshErroredNumCh = make(chan struct{}, synced.PreallocationBatchSize) // Failed refreshes counter channel
)

type Refresher interface {
	RunRefresher()
}

// Refresh is responsible for background refreshing of cache entries.
// It periodically samples random shards and randomly selects "cold" entries
// (from the end of each shard's LRU list) to refresh if necessary.
type Refresh struct {
	ctx             context.Context
	cfg             *config.Config
	balancer        *Balance
	shardsSamplesCh chan *ShardNode
	refreshRespCh   chan *model.Response
	rate            rate.Limiter
}

// NewRefresher constructs a Refresh.
func NewRefresher(ctx context.Context, cfg *config.Config, balancer *Balance) *Refresh {
	return &Refresh{
		ctx:             ctx,
		cfg:             cfg,
		balancer:        balancer,
		shardsSamplesCh: make(chan *ShardNode),
		refreshRespCh:   make(chan *model.Response, rateLimit),
		rate:            rate.NewLimiter(ctx, rateLimit, 0),
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
		r.spawnShardsSamplesProvider()
		for node := range r.shardsSamplesCh {
			r.provideRespRefreshSignal(node)
		}
	}()
}

// spawnShardsSamplesProvider continuously pushes random shard nodes into shardsSamplesCh for processing.
// The channel is closed on shutdown.
func (r *Refresh) spawnShardsSamplesProvider() {
	go func() {
		defer close(r.shardsSamplesCh)
		for {
			select {
			case <-r.ctx.Done():
				return
			case <-r.rate.Chan(): // Throttling (max 1000 checks per second)
				select {
				case <-r.ctx.Done():
					return
				case r.shardsSamplesCh <- r.balancer.RandShardNode():
				}
			}
		}
	}()
}

// update attempts to refresh the given response via Revalidate.
// If successful, increments the refresh metric (in debug mode); otherwise increments the error metric.
func (r *Refresh) update(resp *model.Response) {
	if err := resp.Revalidate(r.ctx); err != nil {
		log.
			Err(err).
			Str("key", strconv.Itoa(int(resp.Key()))).
			Str("shardKey", strconv.Itoa(int(resp.ShardKey()))).
			Str("query", string(resp.ToQuery())).
			Msg("response update failed")

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

// provideRespRefreshSignal selects up to refreshSamples entries from the end of the given shard's LRU list.
// For each candidate, if ShouldRefresh() returns true and rate limiting allows, triggers an asynchronous refresh.
func (r *Refresh) provideRespRefreshSignal(node *ShardNode) {
	lru := node.lruList
	length := lru.Len()
	if length == 0 {
		return
	}

	// Uses round-robbin algo. when storage len is too short for sampling.
	if length <= refreshSamples*10 {
		elem := lru.Back()
		for elem != nil {
			if elem.Value != nil && elem.Value.ShouldRefresh() {
				go r.update(elem.Value)
			}
			elem = elem.Prev()
		}
		return
	}

	// Uses sampling when storage len reached the threshold (refreshSamples*100=~320).
	// Sampling algo. will be more effective from the threshold.
	for i := 0; i < refreshSamples; i++ {
		elem := lru.Back()
		offset := rand.Intn(length)
		for j := 0; j < offset && elem != nil; j++ {
			elem = elem.Prev()
		}
		if elem == nil || elem.Value == nil {
			continue
		}
		if elem.Value.ShouldRefresh() {
			go r.update(elem.Value)
		}
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
