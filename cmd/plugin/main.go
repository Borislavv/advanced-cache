package main

import (
	"container/list"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/algo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"time"
)
import _ "net/http/pprof"
import "net/http"

var reqs []*model.Request
var store *storage.AlgoStorage

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(os.Stdout)

	store = storage.New(config.Storage{
		EvictionAlgo:               string(algo.LRU),
		MemoryFillThreshold:        0.9,
		MemoryLimit:                1024 * 1024 * 10,
		ParallelEvictionsAvailable: 10,
	}, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqs = make([]*model.Request, 0, 10000)
	for i := 0; i < 10000; i++ {
		req := model.NewRequest("285", "1xbet.com", "en", `{"name": "betting", "choice": null}`+strconv.Itoa(i))
		resp, err := model.NewResponse(&list.Element{}, req, []byte(`{"data": "success"}`), time.Minute*10)
		if err != nil {
			panic(err)
		}
		store.Set(ctx, resp)
		reqs = append(reqs, req)
	}
}

func main() {
	go func() {
		if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	var i int
	for {
		if i >= 10000 {
			i = 0
		}
		store.Get(reqs[i%10000])
		i++
	}
}
