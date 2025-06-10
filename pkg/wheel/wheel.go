package wheel

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/rate"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	model "github.com/Borislavv/traefik-http-cache-plugin/pkg/wheel/model"
	"github.com/rs/zerolog/log"
	"strconv"
	"time"
)

const numOfRefreshesPerSec = 10

var (
	refreshedNumCh = make(chan struct{}, synced.PreallocationBatchSize)
	erroredNumCh   = make(chan struct{}, synced.PreallocationBatchSize)
)

type OfTime[T model.Spoke] struct {
	ctx        context.Context
	cfg        *config.Config
	rate       rate.Limiter
	spokesList *list.LockFreeDoublyLinkedList[model.Spoke]
	checkCh    <-chan time.Time
	updateCh   chan model.Spoke
	removeCh   chan *list.LockFreeDoublyLinkedListElement[model.Spoke]
}

func New[T model.Spoke](ctx context.Context, cfg *config.Config) *OfTime[T] {
	w := &OfTime[T]{
		ctx:        ctx,
		cfg:        cfg,
		spokesList: list.NewLockFreeDoublyLinkedList[model.Spoke](),
		checkCh:    utils.NewTicker(ctx, time.Second),
		updateCh:   make(chan model.Spoke, sharded.ShardCount),
		removeCh:   make(chan *list.LockFreeDoublyLinkedListElement[model.Spoke], sharded.ShardCount),
		rate:       rate.NewLimiter(ctx, numOfRefreshesPerSec, numOfRefreshesPerSec),
	}

	w.spawnEventLoop()
	if cfg.IsDebugOn() {
		w.runLogDebugInfo()
	}

	return w
}

func (w *OfTime[T]) Add(spoke T) {
	spoke.StoreWheelListElement(w.spokesList.PushFront(spoke))
}

func (w *OfTime[T]) Touch(spoke T) {
	w.spokesList.MoveToFront(spoke.WheelListElement())
}

func (w *OfTime[T]) Remove(spoke T) {
	w.spokesList.Remove(spoke.WheelListElement())
}

func (w *OfTime[T]) update(spoke model.Spoke) {
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

func (w *OfTime[T]) spawnEventLoop() {
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

func (w *OfTime[T]) runLogDebugInfo() {
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
