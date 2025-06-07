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
	"time"
	"unsafe"
)

type Wheeler interface {
	Add(key uint64, shard uint, interval time.Duration)
}

type Wheel struct {
	ctx        context.Context
	rate       rate.Limiter
	spokesList *list.List[*spoke]
	shardedMap *sharded.Map[*model.Response]
	spokesPool *synced.BatchPool[*spoke]
	checkTimer *time.Timer
	checkCh    <-chan time.Time
	insertCh   chan *spoke
	updateCh   chan *spoke
	removeCh   chan *spoke
}

func (w *Wheel) Memory() uintptr {
	mem := unsafe.Sizeof(w)
	mem += uintptr(w.spokesList.Len() * consts.PtrBytesWeigh)
	mem += uintptr(w.spokesPool.Len() * consts.PtrBytesWeigh)
	return mem
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
		insertCh:   make(chan *spoke, sharded.ShardCount),
		updateCh:   make(chan *spoke, sharded.ShardCount),
		removeCh:   make(chan *spoke, sharded.ShardCount),
		spokesList: list.New[*spoke](isListMustBeThreadSafe),
		spokesPool: synced.NewBatchPool[*spoke](synced.PreallocationBatchSize, func() *spoke {
			return new(spoke)
		}),
		rate: rate.NewLimiter(ctx, numOfRefreshesPerSec, numOfRefreshesPerSec),
	}
	w.spawnEventLoop()
	return w
}

func (w *Wheel) Add(key uint64, shardKey uint64, interval time.Duration) {
	s := w.spokesPool.Get()

	s.key = key
	s.shardKey = shardKey
	s.interval = interval

	w.insertCh <- s
}

func (w *Wheel) spawnEventLoop() {
	w.spawnCheckLoop()
	w.spawnInsertLoop()
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

func (w *Wheel) spawnInsertLoop() {
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case s := <-w.insertCh:
				w.insert(s)
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
			case s := <-w.updateCh:
				w.update(s)
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
			case s := <-w.removeCh:
				w.remove(s)
			}
		}
	}()
}

func (w *Wheel) remove(s *spoke) {
	w.spokesList.Remove(s.element)
	w.spokesPool.Put(s)
}

func (w *Wheel) update(s *spoke) {
	w.rate.Take()
	go func() {
		resp, found := w.shardedMap.Get(s.key, s.shardKey)
		if !found {
			w.removeCh <- s
			return
		}

		if err := resp.Revalidate(w.ctx); err != nil {
			log.
				Err(err).
				Str("key", strconv.Itoa(int(resp.GetRequest().UniqueKey()))).
				Str("shardKey", strconv.Itoa(int(resp.GetShardKey()))).
				Str("query", string(resp.GetRequest().ToQuery())).
				Msg("response update failed")
			return
		}

		s.mustBeRefreshedAt = time.Now().Add(s.interval)
	}()
}

func (w *Wheel) insert(spoke *spoke) {
	spoke.element = w.spokesList.PushFront(spoke)
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
