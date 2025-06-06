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
	//_ "net/http/pprof"
	"runtime"
	"time"
)

func init() {
	if err := godotenv.Overload(".env", ".env.local"); err != nil {
		panic(err)
	}

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

	//go func() {
	//	_ = http.ListenAndServe("localhost:6060", nil)
	//}()
}

func setMaxProcs() {
	if _, err := maxprocs.Set(); err != nil {
		log.Err(err).Msg("setting up GOMAXPROCS value failed")
		panic(err)
	}
	log.Info().Msgf("optimized GOMAXPROCS=%d was sat up", runtime.GOMAXPROCS(0))
}

func loadCfg() *config.Config {
	cfg := &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Err(err).Msg("failed to unmarshal config from envs")
		panic(err)
	}
	cfg.RefreshEvictionDurationThreshold = time.Duration(float64(cfg.RevalidateInterval) * cfg.RevalidateBeta)
	return cfg
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setMaxProcs()
	cfg := loadCfg()
	gc := shutdown.NewGraceful(ctx, cancel)
	gc.SetGracefulTimeout(time.Second * 10)
	probe := liveness.NewProbe(cfg.LivenessProbeTimeout)

	if app, err := cache.NewApp(ctx, cfg, probe); err != nil {
		log.Err(err).Msg("failed init. cache app")
	} else {
		gc.Add(1)
		go app.Start(gc)
	}

	if err := gc.ListenCancelAndAwait(); err != nil {
		log.Err(err).Msg("failed to gracefully shut down service")
	}
}
