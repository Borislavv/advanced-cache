package rate

import (
	"context"
	"errors"
	"gitlab.xbet.lan/v3group/backend/applications/cms/watcher/internal/shared/infrastructure/utils"
	"gitlab.xbet.lan/v3group/backend/packages/go/logger/pkg/logger"
	loggerenum "gitlab.xbet.lan/v3group/backend/packages/go/logger/pkg/logger/enum"
	"runtime"
	"time"
)

type Limiter interface {
	Take() (token struct{})
}

type RateLimiter struct {
	ctx context.Context
	q   chan struct{}
}

type CancelFunc = func()

// NewLimiter - limit: tokens per second will be allocated, init: predefined (allocated) number of tokens (will be allowed on start).
func NewLimiter(ctx context.Context, limit, init int) (*RateLimiter, CancelFunc, error) {
	if init > limit {
		return nil, func() {}, errors.New("init value is greater than limit")
	} else if limit <= 0 {
		return nil, func() {}, errors.New("limit is zero or negative value, potentially dangerous code (deadlock possible)")
	} else if init < 0 {
		logger.JsonRawLog("NewLimiter: init argument was skipped because it has a negative value which is not allowed", loggerenum.WarningLvl, nil)
		init = 0
	}

	ctx, cancel := context.WithCancel(ctx)
	rl := &RateLimiter{ctx: ctx, q: spawnTokenProvider(ctx, limit, init)}

	return rl, func() { cancel() }, nil
}

func (rl *RateLimiter) Take() (token struct{}) {
	return <-rl.q
}

func spawnTokenProvider(ctx context.Context, limit, init int) chan struct{} {
	q := make(chan struct{}, limit)
	for i := 0; i < init; i++ {
		q <- struct{}{}
	}

	go func() {
		defer close(q)

		t := utils.NewTicker(ctx, time.Duration(float64(time.Second)/float64(limit)))
		for {
			select {
			case <-ctx.Done():
				return
			case <-t:
				q <- struct{}{}
			default:
				runtime.Gosched()
			}
		}
	}()

	return q
}
