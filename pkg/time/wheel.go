package time

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/consts"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/rate"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/rs/zerolog/log"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"
)

type Wheeler interface {
	Add(key uint64, shard uint, interval time.Duration)
}

type Wheel struct {
	ctx        context.Context
	rate       rate.Limiter
	spokesList *list.List[*Spoke]
	shardedMap *sharded.Map[*model.Response]
	spokesPool *synced.BatchPool[*Spoke]
	checkTimer *time.Timer
	checkCh    <-chan time.Time
	insertCh   chan *Spoke
	updateCh   chan *Spoke
	removeCh   chan *Spoke
}

func NewWheel(ctx context.Context, shardedMap *sharded.Map[*model.Response]) *Wheel {
	const (
		isListMustBeThreadSafe bool = true
		numOfRefreshesPerSec        = 1000
	)

	w := &Wheel{
		ctx:        ctx,
		shardedMap: shardedMap,
		checkTimer: time.NewTimer(time.Millisecond * 900),
		checkCh:    utils.NewTicker(ctx, time.Second),
		insertCh:   make(chan *Spoke, sharded.ShardCount),
		updateCh:   make(chan *Spoke, sharded.ShardCount),
		removeCh:   make(chan *Spoke, sharded.ShardCount),
		spokesList: list.New[*Spoke](isListMustBeThreadSafe),
		spokesPool: synced.NewBatchPool[*Spoke](synced.PreallocationBatchSize, func() *Spoke {
			return &Spoke{element: &atomic.Pointer[list.Element[*Spoke]]{}}
		}),
		rate: rate.NewLimiter(ctx, numOfRefreshesPerSec, numOfRefreshesPerSec),
	}
	w.spawnEventLoop()
	return w
}

func (w *Wheel) Add(key uint64, shardKey uint64, interval time.Duration) {
	spoke := w.spokesPool.Get()

	atomic.StoreUint64(&spoke.key, key)
	atomic.StoreUint64(&spoke.shardKey, shardKey)
	atomic.StoreInt64(&spoke.interval, interval.Nanoseconds())

	el := w.spokesList.PushFront(spoke)
	spoke.element.Store(el)
}

func (w *Wheel) Memory() uintptr {
	mem := unsafe.Sizeof(w)
	mem += uintptr(w.spokesList.Len() * consts.PtrBytesWeight)
	mem += uintptr(w.spokesPool.Len() * consts.PtrBytesWeight)
	return mem
}

func (w *Wheel) spawnEventLoop() {
	w.spawnCheckLoop()
	w.spawnUpdateLoop()
	w.spawnRemoveLoop()
}

func (w *Wheel) spawnCheckLoop() {
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case <-w.checkCh:
				w.check()
			}
		}
	}()
}

func (w *Wheel) spawnUpdateLoop() {
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case spoke := <-w.updateCh:
				w.update(spoke)
			}
		}
	}()
}

func (w *Wheel) spawnRemoveLoop() {
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case spoke := <-w.removeCh:
				w.remove(spoke)
			}
		}
	}()
}

func (w *Wheel) remove(spoke *Spoke) {
	w.spokesList.Remove(spoke.element.Load())
	w.spokesPool.Put(spoke)
}

func (w *Wheel) update(spoke *Spoke) {
	w.rate.Take()
	go func() {
		resp, releaser, found := w.shardedMap.Get(spoke.key, spoke.shardKey)
		defer releaser.Release()
		if !found {
			w.removeCh <- spoke
			return
		}

		if err := resp.Revalidate(w.ctx); err != nil {
			log.
				Err(err).
				Str("key", strconv.Itoa(int(resp.GetRequest().Key()))).
				Str("shardKey", strconv.Itoa(int(resp.GetRequest().ShardKey()))).
				Str("query", string(resp.GetRequest().ToQuery())).
				Msg("response update failed")
			return
		}

		spoke.mustBeRefreshedAt = time.Now().Add(time.Duration(atomic.LoadInt64(&spoke.interval))).UnixNano()
	}()
}

func (w *Wheel) check() {
	w.checkTimer.Reset(time.Millisecond * 900)

	current := w.spokesList.Back()
	if current == nil {
		return
	}

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.checkTimer.C:
			return
		default:
			if !current.Value.IsReady() {
				return
			}
			w.updateCh <- current.Value
			current = current.Next()
		}
	}
}
