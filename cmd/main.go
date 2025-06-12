package main

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/config"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.xbet.lan/v3group/backend/packages/go/graceful-shutdown/pkg/shutdown"
	"gitlab.xbet.lan/v3group/backend/packages/go/liveness-prober"
	"go.uber.org/automaxprocs/maxprocs"
	"runtime"
	"time"
)

// Initializes environment variables from .env files and binds them using Viper.
// This allows overriding any value via environment variables.
func init() {
	// Load .env and .env.local files for configuration overrides.
	if err := godotenv.Overload(".env", ".env.local"); err != nil {
		panic(err)
	}

	// Bind all relevant environment variables using Viper.
	viper.AutomaticEnv()
	_ = viper.BindEnv("APP_ENV")
	_ = viper.BindEnv("APP_DEBUG")
	_ = viper.BindEnv("INIT_STORAGE_LEN_PER_SHARD")
	_ = viper.BindEnv("EVICTION_ALGO")
	_ = viper.BindEnv("MEMORY_FILL_THRESHOLD")
	_ = viper.BindEnv("MEMORY_LIMIT")
	_ = viper.BindEnv("REVALIDATE_BETA")
	_ = viper.BindEnv("REVALIDATE_INTERVAL")
	_ = viper.BindEnv("SEO_URL")
	_ = viper.BindEnv("SERVER_NAME")
	_ = viper.BindEnv("SERVER_PORT")
	_ = viper.BindEnv("SERVER_SHUTDOWN_TIMEOUT")
	_ = viper.BindEnv("SERVER_REQUEST_TIMEOUT")
	_ = viper.BindEnv("IS_PROMETHEUS_METRICS_ENABLED")
	_ = viper.BindEnv("LIVENESS_PROBE_FAILED_TIMEOUT")
}

// setMaxProcs automatically sets the optimal GOMAXPROCS value (CPU parallelism)
// based on the available CPUs and cgroup/docker CPU quotas (uses automaxprocs).
func setMaxProcs() {
	if _, err := maxprocs.Set(); err != nil {
		log.Err(err).Msg("[main] setting up GOMAXPROCS value failed")
		panic(err)
	}
	log.Info().Msgf("[main] optimized GOMAXPROCS=%d was set up", runtime.GOMAXPROCS(0))
}

// loadCfg loads the configuration struct from environment variables
// and computes any derived configuration values.
func loadCfg() *config.Config {
	cfg := &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Err(err).Msg("[main] failed to unmarshal config from envs")
		panic(err)
	}
	// Calculate the refresh duration threshold as a function of revalidate interval and beta.
	cfg.RefreshDurationThreshold = time.Duration(float64(cfg.RevalidateInterval) * cfg.RevalidateBeta)
	return cfg
}

// Main entrypoint: configures and starts the cache application.
func main() {
	// Create a root context for gracefulShutdown shutdown and cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Optimize GOMAXPROCS for the current environment.
	setMaxProcs()

	// Load the application configuration from env vars.
	cfg := loadCfg()

	// Setup gracefulShutdown shutdown handler (SIGTERM, SIGINT, etc).
	gracefulShutdown := shutdown.NewGraceful(ctx, cancel)
	gracefulShutdown.SetGracefulTimeout(time.Second * 10)

	// Initialize liveness probe for Kubernetes/Cloud health checks.
	probe := liveness.NewProbe(cfg.LivenessProbeTimeout)

	// Initialize and start the cache application.
	if app, err := cache.NewApp(ctx, cfg, probe); err != nil {
		log.Err(err).Msg("[main] failed to init cache app")
	} else {
		// Register app for gracefulShutdown shutdown.
		gracefulShutdown.Add(1)
		go app.Start(gracefulShutdown)
	}

	// Listen for OS signals or context cancellation and wait for gracefulShutdown shutdown.
	if err := gracefulShutdown.ListenCancelAndAwait(); err != nil {
		log.Err(err).Msg("[main] failed to gracefully shut down service")
	}
}
