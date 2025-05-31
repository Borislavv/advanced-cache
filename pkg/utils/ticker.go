package utils

import (
	"context"
	"time"
)

func NewTicker(ctx context.Context, interval time.Duration) (ch <-chan time.Time) {
	ctx, cancel := context.WithCancel(ctx)

	tickCh := make(chan time.Time, 1)
	tickCh <- time.Now()

	go func() {
		ticker := time.NewTicker(interval)
		defer func() {
			ticker.Stop()
			close(tickCh)
			cancel()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				tickCh <- t
			}
		}
	}()

	return tickCh
}
