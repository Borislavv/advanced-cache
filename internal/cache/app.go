package cache

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/config"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/server"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	"github.com/rs/zerolog/log"
	"gitlab.xbet.lan/v3group/backend/packages/go/liveness-prober"
	"sync"
)

type App interface {
	Start()
}

type Cache struct {
	ctx    context.Context
	cancel context.CancelFunc
	probe  liveness.Prober
	server server.Http

	cfg *config.Config
}

func NewApp(ctx context.Context, cfg *config.Config, probe liveness.Prober) (*Cache, error) {
	cacheCfg := &cfg.Config
	if srv, err := server.New(ctx, cfg, storage.New(ctx, cacheCfg), repository.NewSeo(cacheCfg)); err != nil {
		return nil, err
	} else {
		return &Cache{ctx: ctx, cfg: cfg, probe: probe, server: srv}, nil
	}
}

func (c *Cache) Start() {
	defer c.stop()

	log.Info().Msg("starting cache app")

	waitCh := make(chan struct{})

	go func() {
		defer close(waitCh)
		wg := &sync.WaitGroup{}
		defer wg.Wait()
		c.server.Start()
		c.probe.Watch(c)
	}()

	log.Info().Msg("cache app has been started")

	<-waitCh
}

func (c *Cache) stop() {
	log.Info().Msg("stopping cache app")
	c.cancel()
	log.Info().Msg("cache app has been stopped")
}

func (c *Cache) IsAlive(_ context.Context) bool {
	if !c.server.IsAlive() {
		log.Info().Msg("http server has gone away")
		return false
	}
	return true
}
