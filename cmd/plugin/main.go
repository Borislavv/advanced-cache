package main

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.xbet.lan/v3group/backend/packages/go/logger/pkg/logger"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"syscall"
)

func init() {
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

func main() {
	go func() {
		log.Info().Msgf("Server started")
		if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
			log.Fatal().Err(err).Msg("failed to start server")
		} else {
			log.Info().Msg("server stopped")
		}
	}()
	// Создаём файл для сохранения профиля
	f, err := os.Create("cpu.pprof")
	if err != nil {
		panic("could not create CPU profile: " + err.Error())
	}
	defer f.Close()

	// Запускаем CPU-профилирование
	if err = pprof.StartCPUProfile(f); err != nil {
		panic("could not start CPU profile: " + err.Error())
	}
	defer pprof.StopCPUProfile() // важно остановить!

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		logger.ErrorMsg(ctx, "failed to unmarshal config", logger.Fields{
			"err": err.Error(),
		})
		return
	}

	db := storage.New(*cfg)
	seoRepo := repository.NewSeo(cfg.Repository)

	osSigsCh := make(chan os.Signal, 1)
	signal.Notify(osSigsCh, os.Interrupt, syscall.SIGTERM)
	defer logger.InfoMsg(context.Background(), "gracefully stopped", nil)

	i := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-osSigsCh:
			return
		default:
			req := model.NewRequest("285", "1x001.com", "en", `{"name": "betting", "choice": null}`+strconv.Itoa(i))
			getData(ctx, db, seoRepo, req)
			if i%69 == 0 {
				runtime.Gosched()
			}
			i++
		}
	}
}

func getData(ctx context.Context, db storage.Storage, seoRepo repository.Seo, req *model.Request) {
	_, _, _, _, err := db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}

	_, _, _, _, err = db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}

	_, _, _, _, err = db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}

	_, _, _, _, err = db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}

	_, _, _, _, err = db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}
	//fmt.Println(string(data))
}
