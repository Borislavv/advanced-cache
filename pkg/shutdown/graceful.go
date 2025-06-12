package shutdown

import (
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var TimeoutError = errors.New("timeout error")

type Gracefuller interface {
	// Add is a clone of (sync.WaitGroup).Add() method which must be called on main goroutines.
	Add(n int)
	// Done is a clone of (sync.WaitGroup).Done() method which must be called on closing main goroutines.
	Done()
	// SetGracefulTimeout set a time and this time will be used as a timeout when context will be closed while
	// all goroutines cancelling (by default: 10s.).
	SetGracefulTimeout(timeout time.Duration)
	// ListenCancelAndAwait will catch one of channels (osSigsCh:[syscall.SIGINT, syscall.SIGTERM])
	// or ctx.Done() and awaits while all main goroutines will be finished by sync.WaitGroup.
	// NOTE: the ListenCancelAndAwait method is a synchronous (blocking) and must not be called from goroutine.
	// Also, must be called at the end of the main function.
	ListenCancelAndAwait() error
}

type Graceful struct {
	wg       *sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	timeout  time.Duration
	osSigsCh chan os.Signal
}

// NewGraceful is a constructor of new Graceful shutdown implementation.
// Accepts main context.Context and context.CancelFunc.
func NewGraceful(ctx context.Context, cancel context.CancelFunc) *Graceful {
	gsh := &Graceful{
		wg:       &sync.WaitGroup{},
		ctx:      ctx,
		cancel:   cancel,
		timeout:  time.Second * 10,
		osSigsCh: make(chan os.Signal, 1),
	}

	signal.Notify(gsh.osSigsCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	return gsh
}

// Add is a clone of (sync.WaitGroup).Add() method which must be called on main goroutines.
func (g *Graceful) Add(n int) {
	g.wg.Add(n)
}

// Done is a clone of (sync.WaitGroup).Done() method which must be called on closing main goroutines.
func (g *Graceful) Done() {
	g.wg.Done()
}

// SetGracefulTimeout set a time and this time will be used as a timeout when context will be closed while
// all goroutines cancelling (by default: 10s.).
func (g *Graceful) SetGracefulTimeout(timeout time.Duration) {
	g.timeout = timeout
}

// ListenCancelAndAwait will catch one of channels (osSigsCh:[syscall.SIGINT, syscall.SIGTERM])
// or ctx.Done() and awaits while all main goroutines will be finished by sync.WaitGroup.
// NOTE: the ListenCancelAndAwait method is a synchronous (blocking) and must not be called from goroutine.
// Also, must be called at the end of the main function.
func (g *Graceful) ListenCancelAndAwait() error {
	select {
	case <-g.ctx.Done():
		log.Info().Msg("[graceful-shutdown] received a ctx.Done() signal, starting cancellation")
	case sig := <-g.osSigsCh:
		log.Info().Msgf("[graceful-shutdown] received a %v signal, starting cancellation", sig.String())
	}
	return g.cancelAndAwaitWithTimeout()
}

func (g *Graceful) cancelAndAwaitWithTimeout() error {
	// cancel context of all application
	g.cancel()

	// timeout timer
	ttrCh := time.NewTimer(g.timeout)
	defer ttrCh.Stop()

	// is success channel
	sucCh := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(sucCh)
	}()

	// result channel
	errCh := make(chan error)

	go func() {
		// this goroutine is writer-side of channel and correctly close the channel is here
		defer close(errCh)

		select {
		case <-sucCh:
			log.Info().Msg("[graceful-shutdown] service was gracefully shut down")
			errCh <- nil
		case <-ttrCh.C:
			log.Info().Msgf("[graceful-shutdown] %s, not all goroutines were closed within %s", TimeoutError.Error(), g.timeout.String())
			errCh <- TimeoutError
		}
	}()

	return <-errCh
}
