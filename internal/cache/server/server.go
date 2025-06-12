package server

import (
	"context"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/api"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/rs/zerolog/log"
	"gitlab.xbet.lan/v3group/backend/packages/go/httpserver/pkg/httpserver"
	"gitlab.xbet.lan/v3group/backend/packages/go/httpserver/pkg/httpserver/controller"
	"gitlab.xbet.lan/v3group/backend/packages/go/httpserver/pkg/httpserver/middleware"
	"gitlab.xbet.lan/v3group/backend/packages/go/liveness-prober"
	"gitlab.xbet.lan/v3group/backend/packages/go/metrics"
	api2 "gitlab.xbet.lan/v3group/backend/packages/go/metrics/controller"
	prometheusrequestmiddleware "gitlab.xbet.lan/v3group/backend/packages/go/metrics/middleware"
	"sync"
	"sync/atomic"
)

// Error messages used for server and metrics initialization.
var (
	InitFailedErrorMessage        = "[server] init. failed"
	MetricsInitFailedErrorMessage = "[server] init. prometheus metrics failed"
)

// Http interface exposes methods for starting and liveness probing.
type Http interface {
	Start()
	IsAlive() bool
}

// HttpServer implements Http, wraps all dependencies required for running the HTTP server.
type HttpServer struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg           *config.Config
	metrics       *metrics.Metrics
	server        *httpserver.HTTP
	isServerAlive *atomic.Bool
	db            storage.Storage
}

// New creates a new HttpServer, initializing metrics and the HTTP server itself.
// If any step fails, returns an error and performs cleanup.
func New(
	ctx context.Context,
	cfg *config.Config,
	db storage.Storage,
	seoRepo repository.Backender,
	reader synced.PooledReader,
	probe liveness.Prober,
) (*HttpServer, error) {
	var err error

	// Create cancellable context for graceful shutdown.
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	srv := &HttpServer{
		ctx:           ctx,
		cancel:        cancel,
		cfg:           cfg,
		db:            db,
		isServerAlive: &atomic.Bool{},
	}

	// Initialize Prometheus or other metrics.
	if err = srv.initMetrics(); err != nil {
		log.Err(err).Msg(MetricsInitFailedErrorMessage)
		return nil, errors.New(MetricsInitFailedErrorMessage)
	}

	// Initialize HTTP server with all controllers and middlewares.
	if err = srv.initServer(db, seoRepo, reader, probe); err != nil {
		log.Err(err).Msg(InitFailedErrorMessage)
		return nil, errors.New(InitFailedErrorMessage)
	}

	return srv, nil
}

// Start runs the HTTP server in a goroutine and waits for it to finish.
func (s *HttpServer) Start() {
	defer s.stop()

	waitCh := make(chan struct{})

	go func() {
		defer close(waitCh)
		wg := &sync.WaitGroup{}
		defer wg.Wait()
		s.spawnServer(wg)
	}()

	<-waitCh
}

// stop cancels the context, signaling shutdown to all server goroutines.
func (s *HttpServer) stop() {
	s.cancel()
	s.db.Stop()
}

// IsAlive returns true if the server is marked as alive.
func (s *HttpServer) IsAlive() bool {
	return s.isServerAlive.Load()
}

// spawnServer starts the HTTP server in a new goroutine, sets server liveness flags, and blocks until it exits.
func (s *HttpServer) spawnServer(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer func() {
			s.isServerAlive.Store(false)
			wg.Done()
		}()
		s.isServerAlive.Store(true)
		s.server.ListenAndServe()
	}()
}

// initCtx allows updating/replacing the server's context and cancel func.
func (s *HttpServer) initCtx(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.ctx = ctx
	s.cancel = cancel
}

// initMetrics initializes Prometheus (or custom) metrics registry and binds it to the server.
func (s *HttpServer) initMetrics() error {
	prometheusMetrics, err := metrics.New()
	if err != nil {
		log.Err(err).Msg(MetricsInitFailedErrorMessage)
		return errors.New(MetricsInitFailedErrorMessage)
	}
	s.metrics = prometheusMetrics
	return nil
}

// initServer creates the HTTP server instance, sets up controllers and middlewares, and stores the result.
func (s *HttpServer) initServer(
	cache storage.Storage,
	seoRepo repository.Backender,
	reader synced.PooledReader,
	probe liveness.Prober,
) error {
	ctx, cancel := context.WithCancel(s.ctx)
	s.cancel = cancel

	// Compose server with controllers and middlewares.
	if server, err := httpserver.New(ctx, s.cfg, s.controllers(cache, seoRepo, reader, probe), s.middlewares()); err != nil {
		cancel()
		log.Err(err).Msg(InitFailedErrorMessage)
		return errors.New(InitFailedErrorMessage)
	} else {
		s.server = server
	}

	return nil
}

// controllers returns all HTTP controllers for the server (endpoints/handlers).
func (s *HttpServer) controllers(
	cache storage.Storage,
	seoRepo repository.Backender,
	reader synced.PooledReader,
	probe liveness.Prober,
) []controller.HttpController {
	return []controller.HttpController{
		api.NewLivenessController(probe),                             // Liveness/healthcheck endpoint
		api.NewCacheController(s.ctx, s.cfg, cache, seoRepo, reader), // Main cache handler
		api2.NewPrometheusMetrics(s.ctx),                             // Prometheus metrics endpoint
	}
}

// middlewares returns the request middlewares for the server, executed in reverse order.
func (s *HttpServer) middlewares() []middleware.HttpMiddleware {
	return []middleware.HttpMiddleware{
		/** exec 1st. */ middleware.NewApplicationJsonMiddleware(), // Sets Content-Type
		/** exec 2nd. */ middleware.NewWatermarkMiddleware(s.ctx, s.cfg), // Adds watermark info
		/** exec 3rd. */ prometheusrequestmiddleware.NewPrometheusMetrics(s.ctx, s.metrics), // Prometheus instrumentation
	}
}
