package rate

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"time"
)

// Limiter defines the interface for a rate limiter with both blocking (Take) and non-blocking (Chan) API.
type Limiter interface {
	// Take waits for a token (or until ctx.Done), returns (token, true) on success or (zero, false) if ctx is cancelled.
	Take(ctx context.Context) (token struct{}, ok bool)
	// Chan returns the underlying channel for non-blocking/asynchronous usage.
	Chan() <-chan struct{}
}

// Limit is a token bucket rate limiter.
// It dispenses up to "limit" tokens per second, with optional initial burst size "init".
type Limit struct {
	ctx context.Context // Limiter's parent context
	q   chan struct{}   // Token channel (buffered to 'limit')
}

// NewLimiter constructs a new rate limiter.
//   - limit: number of tokens per second (rate).
//   - init: initial tokens available immediately (burst).
func NewLimiter(ctx context.Context, limit, init int) *Limit {
	return &Limit{ctx: ctx, q: spawnTokenProvider(ctx, limit, init)}
}

// Chan returns the token channel for advanced usage (select etc).
func (rl *Limit) Chan() <-chan struct{} {
	return rl.q
}

// Take attempts to get a token, blocking until a token is available or ctx is done.
// If ctx is nil, uses the limiter's parent context.
func (rl *Limit) Take(ctx context.Context) (token struct{}, ok bool) {
	if ctx == nil {
		ctx = rl.ctx
	}
	select {
	case <-ctx.Done():
		return token, false
	case s := <-rl.q:
		return s, true
	}
}

// spawnTokenProvider launches a goroutine that fills the token channel at a fixed rate.
func spawnTokenProvider(ctx context.Context, limit, init int) chan struct{} {
	q := make(chan struct{}, limit)

	// Initial burst: fill the channel with 'init' tokens immediately
	for i := 0; i < init; i++ {
		q <- struct{}{}
	}

	go func() {
		defer close(q)
		// Ticker for rate limiting; period = 1/limit seconds per token
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
