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
	"gitlab.xbet.lan/v3group/backend/packages/go/metrics"
	"sync"
	"sync/atomic"
)

var (
	InitFailedErrorMessage        = "server init. failed"
	MetricsInitFailedErrorMessage = "failed to init. prometheus metrics"
)

type Http interface {
	Start()
	IsAlive() bool
}

type HttpServer struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg           *config.Config
	metrics       *metrics.Metrics
	server        *httpserver.HTTP
	isServerAlive *atomic.Bool
}

func New(
	ctx context.Context,
	cfg *config.Config,
	cache storage.Storage,
	seoRepo repository.Seo,
	reader synced.PooledReader,
) (*HttpServer, error) {
	var err error

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
		isServerAlive: &atomic.Bool{},
	}

	if err = srv.initMetrics(); err != nil {
		log.Err(err).Msg("metrics init. failed")
		return nil, errors.New(MetricsInitFailedErrorMessage)
	}

	if err = srv.initServer(cache, seoRepo, reader); err != nil {
		log.Err(err).Msg("server init. failed")
		return nil, errors.New(InitFailedErrorMessage)
	}

	return srv, nil
}

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

func (s *HttpServer) stop() {
	s.cancel()
}

func (s *HttpServer) IsAlive() bool {
	return s.isServerAlive.Load()
}

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

func (s *HttpServer) initCtx(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.ctx = ctx
	s.cancel = cancel
}

func (s *HttpServer) initMetrics() error {
	prometheusMetrics, err := metrics.New()
	if err != nil {
		log.Err(err).Msg(MetricsInitFailedErrorMessage)
		return errors.New(MetricsInitFailedErrorMessage)
	}
	s.metrics = prometheusMetrics
	return nil
}

func (s *HttpServer) initServer(cache storage.Storage, seoRepo repository.Seo, reader synced.PooledReader) error {
	ctx, cancel := context.WithCancel(s.ctx)
	s.cancel = cancel

	if server, err := httpserver.New(ctx, s.cfg, s.controllers(cache, seoRepo, reader), s.middlewares()); err != nil {
		cancel()
		log.Err(err).Msg(InitFailedErrorMessage)
		return errors.New(InitFailedErrorMessage)
	} else {
		s.server = server
	}

	return nil
}

// controllers returns a slice of server.HttpController[s] for http server (handlers).
func (s *HttpServer) controllers(cache storage.Storage, seoRepo repository.Seo, reader synced.PooledReader) []controller.HttpController {
	return []controller.HttpController{
		api.NewCacheController(s.ctx, s.cfg, cache, seoRepo, reader),
	}
}

// requestMiddlewares returns a slice of server.HttpMiddleware[s] which will executes in reverse order before handling request.
func (s *HttpServer) middlewares() []middleware.HttpMiddleware {
	return []middleware.HttpMiddleware{
		/** exec 2nd. */ middleware.NewApplicationJsonMiddleware(),
		/** exec 4th. */ middleware.NewWatermarkMiddleware(s.ctx, s.cfg),
	}
}
