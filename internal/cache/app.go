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
	"github.com/rs/zerolog/log"
	"gitlab.xbet.lan/v3group/backend/packages/go/graceful-shutdown/pkg/shutdown"
	"gitlab.xbet.lan/v3group/backend/packages/go/liveness-prober"
)

// App defines the cache application lifecycle interface.
type App interface {
	Start()
}

// Cache encapsulates the entire cache application state, including HTTP server, config, and probes.
type Cache struct {
	cfg    *config.Config     // Application configuration
	ctx    context.Context    // Application context for cancellation and shutdown
	cancel context.CancelFunc // Cancel function for ctx
	probe  liveness.Prober    // Liveness probe integration
	server server.Http        // HTTP server (implements business logic and API)
}

// NewApp builds a new Cache app, wiring together storage, repo, reader, and server.
func NewApp(ctx context.Context, cfg *config.Config, probe liveness.Prober) (*Cache, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Setup sharded map for high-concurrency cache storage.
	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	store := storage.New(ctx, &cfg.Config, shardedMap)
	reader := synced.NewPooledResponseReader(synced.PreallocationBatchSize)
	repo := repository.NewSeo(&cfg.Config, reader)

	// Compose the HTTP server (API, metrics and so on)
	srv, err := server.New(ctx, cfg, store, repo, reader, probe)
	if err != nil {
		cancel()
		return nil, err
	}

	return &Cache{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		probe:  probe,
		server: srv,
	}, nil
}

// Start runs the cache server and liveness probe, and handles graceful shutdown.
// The Gracefuller interface is expected to call Done() when shutdown is complete.
func (c *Cache) Start(gc shutdown.Gracefuller) {
	defer func() {
		c.stop()
		gc.Done()
	}()

	log.Info().Msg("starting cache app")

	waitCh := make(chan struct{})

	go func() {
		defer close(waitCh)
		c.probe.Watch(c) // Call first due to it does not block the green-thread
		c.server.Start() // Blocks the green-thread
	}()

	log.Info().Msg("cache app has been started")

	<-waitCh // Wait until the server exits
}

// stop cancels the main application context and logs shutdown.
func (c *Cache) stop() {
	log.Info().Msg("stopping cache app")
	c.cancel()
	log.Info().Msg("cache app has been stopped")
}

// IsAlive is called by liveness probes to check app health.
// Returns false if the HTTP server is not alive.
func (c *Cache) IsAlive(_ context.Context) bool {
	if !c.server.IsAlive() {
		log.Info().Msg("http server has gone away")
		return false
	}
	return true
}
