package main

import (
	"context"
	"fmt"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"net/http"
	"os"
)

func InitEnv() {
	_ = godotenv.Load()
	viper.AutomaticEnv()
	_ = viper.BindEnv("INIT_STORAGE_LEN_PER_SHARD")
	_ = viper.BindEnv("EVICTION_ALGO")
	_ = viper.BindEnv("MEMORY_FILL_THRESHOLD")
	_ = viper.BindEnv("MEMORY_LIMIT")
	_ = viper.BindEnv("REVALIDATE_BETA")
	_ = viper.BindEnv("REVALIDATE_INTERVAL")
	_ = viper.BindEnv("SEO_URL")
}

func InitLogger(level zerolog.Level) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(os.Stdout)
}

func init() {
	InitEnv()
	InitLogger(zerolog.DebugLevel)
}

func main() {
	go func() {
		if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Err(err).Msg("failed to unmarshal config")
		return
	}

	db := storage.New(*cfg)
	seoRepo := repository.NewSeo(cfg.Repository)
	req := model.NewRequest("285", "1x001.com", "en", `{"name": "betting", "choice": null}`)

	_, _, _, found, err := db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}
	if !found {
		log.Info().Msg("it's correct, must not be found")
	}

	_, data, _, found, err := db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}
	if !found {
		log.Error().Msg("this is bad, very bad, data is not found")
		return
	}
	fmt.Println(string(data))
}
