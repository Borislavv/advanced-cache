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
	rateLimit      = 512
	refreshSamples = 32 // number of items limit for refresh per shard
)

var (
	refreshedNumCh = make(chan struct{}, synced.PreallocationBatchSize)
	erroredNumCh   = make(chan struct{}, synced.PreallocationBatchSize)
)

// Refresher - refreshes responses by sampling shards and the end of LRU list.
type Refresher struct {
	ctx             context.Context
	cfg             *config.Config
	balancer        *Balancer
	shardsSamplesCh chan *shardNode
	refreshRespCh   chan *model.Response
	rate            rate.Limiter
}

func NewRefresher(ctx context.Context, cfg *config.Config, balancer *Balancer) *Refresher {
	return &Refresher{
		ctx:             ctx,
		cfg:             cfg,
		balancer:        balancer,
		shardsSamplesCh: make(chan *shardNode),
		refreshRespCh:   make(chan *model.Response, rateLimit),
		rate:            rate.NewLimiter(ctx, rateLimit, 0),
	}
}

func (r *Refresher) RunRefresher() {
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

func (r *Refresher) spawnShardsSamplesProvider() {
	go func() {
		defer close(r.shardsSamplesCh)
		for {
			select {
			case <-r.ctx.Done():
				return
			case r.shardsSamplesCh <- r.balancer.randShardNode():
			}
		}
	}()
}

func (r *Refresher) update(resp *model.Response) {
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
			case erroredNumCh <- struct{}{}:
			}
		}
		return
	}

	if r.cfg.IsDebugOn() {
		select {
		case <-r.ctx.Done():
		case refreshedNumCh <- struct{}{}:
		}
	}
}

// provideRespRefreshSignal - provides N random samples from the back of LRU list.
func (r *Refresher) provideRespRefreshSignal(node *shardNode) {
	lru := node.lruList

	length := lru.Len()
	if length == 0 {
		return
	}
	if length-refreshSamples > refreshSamples {
		length -= refreshSamples
	}

	// cold sampling for the end of the list
	for i := 0; i < refreshSamples; i++ {
		elem := lru.Back()
		offset := rand.Intn(length)
		for j := 0; j < offset && elem != nil; j++ {
			elem = elem.Prev()
		}
		if elem == nil || elem.Value == nil {
			continue
		}

		resp := elem.Value
		if resp.ShouldRefresh() {
			select {
			case <-r.ctx.Done():
				return
			case <-r.rate.Chan():
				go r.update(resp)
			}
		}
	}
}

func (r *Refresher) runLogger() {
	go func() {
		refreshesNumPer5Sec := 0
		erroredNumPer5Sec := 0
		ticker := utils.NewTicker(r.ctx, 5*time.Second)
		for {
			select {
			case <-r.ctx.Done():
				return
			case <-refreshedNumCh:
				refreshesNumPer5Sec += 1
			case <-erroredNumCh:
				erroredNumPer5Sec += 1
			case <-ticker:
				var (
					successNum = strconv.Itoa(refreshesNumPer5Sec)
					erroredNum = strconv.Itoa(erroredNumPer5Sec)
				)

				log.
					Info().
					//Str("target", "wheel").
					//Str("processed", successNum).
					//Str("errored", erroredNum).
					Msgf("[refresher][5s] success %s, errors: %s", successNum, erroredNum)

				refreshesNumPer5Sec = 0
				erroredNumPer5Sec = 0
			}
		}
	}()
}
