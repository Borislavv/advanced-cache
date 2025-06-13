package liveness

import (
	"context"
	"github.com/rs/zerolog/log"
	"time"
)

// Prober can handle services/applications.
type Prober interface {
	// Watch starts a goroutine which watches for new messages in the liveness channel and respond.
	// This method must be called on main service thread before the lock action will be called.
	// Example:
	//        livenessProbe := liveness.NewProbe(ctx)
	// 	      usefulService := useful.NewService(ctx)
	//        livenessProbe.Watch(usefulService)
	//        wg.Add(1)
	//        go usefulService.DoWork(wg)
	//        // service is alive and probe will be report it
	// 		  isAlive := livenessProbe.IsAlive() // (bool) true
	//		  wg.Wait()
	// 		  isAlive := livenessProbe.IsAlive() // (bool) false (but depends on service which  respond on IsAlive question.)
	Watch(services ...Service)
	// IsAlive checks wether is target service (a service which called Watch method) alive.
	IsAlive() bool
}

type Prob struct {
	askCh   chan context.Context
	respCh  chan bool
	timeout time.Duration
}

func NewProbe(timeout time.Duration) *Prob {
	if timeout < time.Millisecond {
		log.Err(TimeoutIsTooShortError).Msg("min timeout duration is 1ms " +
			"(timeout sat upped as 10ms as a more reasonable value, " +
			"if you need it shorter, configure it's properly)")
		timeout = time.Millisecond * 10
	}

	return &Prob{
		askCh:   make(chan context.Context),
		respCh:  make(chan bool),
		timeout: timeout,
	}
}

func (p *Prob) Watch(services ...Service) {
	go func() {
		for ctx := range p.askCh {
			isAlive := true
			for _, service := range services {
				isAlive = isAlive && service.IsAlive(ctx)
			}
			p.respCh <- isAlive
		}
	}()
}

func (p *Prob) IsAlive() bool {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	go func() {
		select {
		case p.askCh <- ctx:
		case <-ctx.Done():
			log.Err(ctx.Err()).Msg("liveness probe deadline exceeded while pushing trigger into channel" +
				" (check the IsAlive method use context properly and also the Watch method was called)")
		}
	}()

	select {
	case <-ctx.Done():
		return false
	case r := <-p.respCh:
		return r
	}
}
