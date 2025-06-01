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
	"testing"
	"time"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(os.Stdout)
}

var BenchmarkReadFromClusterNum int

func BenchmarkReadFromCluster(b *testing.B) {
	log.Info().Msg("Started i: " + strconv.Itoa(BenchmarkReadFromClusterNum) +
		" BenchmarkReadFromCluster benchmark with " + strconv.Itoa(b.N) + " iterations.")
	BenchmarkReadFromClusterNum++

	s := storage.New(config.Storage{
		EvictionAlgo:               string(algo.LRU),
		MemoryFillThreshold:        0.95,
		MemoryLimit:                1024 * 1024 * 128,
		ParallelEvictionsAvailable: 2,
	}, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requests := make([]*model.Request, 0, b.N)
	for i := 0; i < b.N; i++ {
		req := model.NewRequest("285", "1xbet.com", "en", `{"name": "betting", "choice": null}`+strconv.Itoa(i))
		resp, err := model.NewResponse(&list.Element{}, req, []byte(`{"data": "success"}`), time.Minute*10)
		if err != nil {
			panic(err)
		}
		s.Set(ctx, resp)
		requests = append(requests, req)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Get(requests[i%b.N])
			i++
		}
	})
}

var BenchmarkWriteIntoClusterNum int

func BenchmarkWriteIntoCluster(b *testing.B) {
	log.Info().Msg("Started i: " + strconv.Itoa(BenchmarkWriteIntoClusterNum) +
		" BenchmarkWriteIntoCluster benchmark with " + strconv.Itoa(b.N) + " iterations.")
	BenchmarkWriteIntoClusterNum++

	s := storage.New(config.Storage{
		EvictionAlgo:               string(algo.LRU),
		MemoryFillThreshold:        0.95,
		MemoryLimit:                1024 * 1024 * 128,
		ParallelEvictionsAvailable: 2,
	}, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	responses := make([]*model.Response, b.N)
	for i := 0; i < b.N; i++ {
		req := model.NewRequest("285", "1xbet.com", "en", `{"name": "betting", "choice": null}`+strconv.Itoa(i))
		resp, err := model.NewResponse(&list.Element{}, req, []byte(`{"data": "success"}`), time.Minute*10)
		if err != nil {
			panic(err)
		}
		responses[i] = resp
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Set(ctx, responses[i%b.N])
			i++
		}
	})
}
