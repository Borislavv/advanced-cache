package cache

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/config"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/server"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/wheel"
	wheelmodel "github.com/Borislavv/traefik-http-cache-plugin/pkg/wheel/model"
	"github.com/rs/zerolog/log"
	"gitlab.xbet.lan/v3group/backend/packages/go/graceful-shutdown/pkg/shutdown"
	"gitlab.xbet.lan/v3group/backend/packages/go/liveness-prober"
)

type App interface {
	Start()
}

type Cache struct {
	cfg    *config.Config
	ctx    context.Context
	cancel context.CancelFunc
	probe  liveness.Prober
	server server.Http
}

func NewApp(ctx context.Context, cfg *config.Config, probe liveness.Prober) (*Cache, error) {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel

	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	timeWheel := wheel.New[wheelmodel.Spoke](ctx, cfg)
	store := storage.New(ctx, &cfg.Config, timeWheel, shardedMap)
	reader := synced.NewPooledResponseReader(synced.PreallocationBatchSize)
	repo := repository.NewSeo(&cfg.Config, reader)

	if srv, err := server.New(ctx, cfg, store, repo, reader); err != nil {
		return nil, err
	} else {
		return &Cache{ctx: ctx, cancel: cancel, cfg: cfg, probe: probe, server: srv}, nil
	}
}

func (c *Cache) Start(gc shutdown.Gracefuller) {
	defer func() {
		c.stop()
		gc.Done()
	}()

	log.Info().Msg("starting cache app")

	waitCh := make(chan struct{})

	go func() {
		defer close(waitCh)
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
