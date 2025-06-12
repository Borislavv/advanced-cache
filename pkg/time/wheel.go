package timed

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/rate"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	timemodel "github.com/Borislavv/traefik-http-cache-plugin/pkg/time/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"strconv"
	"time"
)

const numOfRefreshesPerSec = 10

var (
	refreshedNumCh = make(chan struct{}, synced.PreallocateBatchSize)
	erroredNumCh   = make(chan struct{}, synced.PreallocateBatchSize)
)

type Wheel[T timemodel.Spoke] struct {
	ctx        context.Context
	cfg        *config.Config
	rate       rate.Limiter
	spokesList *list.List[timemodel.Spoke]
	checkCh    <-chan time.Time
	updateCh   chan timemodel.Spoke
	removeCh   chan *list.Element[timemodel.Spoke]
}

func New[T timemodel.Spoke](ctx context.Context, cfg *config.Config) *Wheel[T] {
	w := &Wheel[T]{
		ctx:        ctx,
		cfg:        cfg,
		spokesList: list.New[timemodel.Spoke](true),
		checkCh:    utils.NewTicker(ctx, time.Second),
		updateCh:   make(chan timemodel.Spoke, sharded.ShardCount),
		removeCh:   make(chan *list.Element[timemodel.Spoke], sharded.ShardCount),
		rate:       rate.NewLimiter(ctx, numOfRefreshesPerSec, numOfRefreshesPerSec),
	}

	w.spawnEventLoop()
	if cfg.IsDebugOn() {
		w.runLogDebugInfo()
	}

	return w
}

func (w *Wheel[T]) Add(spoke T) {
	spoke.StoreWheelListElement(w.spokesList.PushFront(spoke))
}

func (w *Wheel[T]) Touch(spoke T) {
	w.spokesList.MoveToFront(spoke.WheelListElement())
}

func (w *Wheel[T]) Remove(spoke T) {
	w.spokesList.Remove(spoke.WheelListElement())
}

func (w *Wheel[T]) update(spoke timemodel.Spoke) {
	if err := spoke.Revalidate(w.ctx); err != nil {
		log.
			Err(err).
			Str("key", strconv.Itoa(int(spoke.Key()))).
			Str("shardKey", strconv.Itoa(int(spoke.ShardKey()))).
			Str("query", string(spoke.ToQuery())).
			Msg("response update failed")
		erroredNumCh <- struct{}{}
		return
	}
	w.spokesList.MoveToFront(spoke.WheelListElement())
	refreshedNumCh <- struct{}{}
}

func (w *Wheel[T]) spawnEventLoop() {
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case <-w.checkCh:
				ctx, cancel := context.WithTimeout(w.ctx, time.Millisecond*800)

				current := w.spokesList.Back()
				if current == nil {
					cancel()
					continue
				}
			loop:
				for {
					select {
					case <-ctx.Done():
						break loop
					default:
						if current == nil {
							break loop
						}
						if current.Value == nil {
							select {
							case <-ctx.Done():
								break loop
							case w.removeCh <- current:
							}
							continue loop
						}
						if current.Value.ShouldRefresh() {
							select {
							case <-ctx.Done():
								break loop
							case <-w.rate.Chan():
								go w.update(current.Value)
							}
						}
						current = current.Next()
					}
				}

				cancel()
			}
		}
	}()
}

func (w *Wheel[T]) runLogDebugInfo() {
	go func() {
		refreshesNumPer5Sec := 0
		erroredNumPer5Sec := 0
		ticker := utils.NewTicker(w.ctx, 5*time.Second)
		for {
			select {
			case <-w.ctx.Done():
				return
			case <-refreshedNumCh:
				refreshesNumPer5Sec += 1
			case <-erroredNumCh:
				erroredNumPer5Sec += 1
			case <-ticker:
				var (
					processedNum = strconv.Itoa(refreshesNumPer5Sec)
					erroredNum   = strconv.Itoa(refreshesNumPer5Sec)
					queueLen     = strconv.Itoa(int(w.spokesList.Len()))
				)

				log.
					Info().
					//Str("target", "wheel").
					//Str("processed", processedNum).
					//Str("errored", erroredNum).
					//Str("queue", queueLen).
					Msgf("[wheel][5s] refreshed %s, errored: %s, queue: %s", processedNum, erroredNum, queueLen)

				refreshesNumPer5Sec = 0
				erroredNumPer5Sec = 0
			}
		}
	}()
}
