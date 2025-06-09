package rate

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"time"
)

type Limiter interface {
	Take() (token struct{})
}

type Limit struct {
	ctx context.Context
	q   chan struct{}
}

// NewLimiter - limit: tokens per second will be allocated, init: predefined (allocated) number of tokens (will be allowed on start).
func NewLimiter(ctx context.Context, limit, init int) *Limit {
	return &Limit{ctx: ctx, q: spawnTokenProvider(ctx, limit, init)}
}

func (rl *Limit) Take() (token struct{}) {
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
			}
		}
	}()

	return q
}
