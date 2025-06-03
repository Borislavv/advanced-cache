package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(os.Stdout)
}

var BenchmarkReadFromStorageNum int

func BenchmarkReadFromStorage(b *testing.B) {
	log.Info().Msg("[" + strconv.Itoa(BenchmarkReadFromStorageNum) + "] Started BenchmarkReadFromStorage benchmark with " + strconv.Itoa(b.N) + " iterations.")
	BenchmarkReadFromStorageNum++

	ctx := context.Background()

	cfg := &config.Config{
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024 * 12,
	}

	db := New(ctx, cfg)

	requests := make([]*model.Request, 0, b.N)
	for i := 0; i < b.N; i++ {
		req := model.NewRequest("285", "1xbet.com", "en", `{"name": "node_`+strconv.Itoa(i)+`", "choice": null}`)
		resp, err := model.NewResponse(
			model.NewData(200, http.Header{}, []byte(`{"data": "success"}`)),
			req,
			cfg,
			func(ctx context.Context) (data *model.Data, err error) {
				return model.NewData(200, http.Header{}, []byte(`{"data": "success"}`)), nil
			},
		)
		if err != nil {
			panic(err)
		}
		resp.Revalidate(ctx)
		db.Set(ctx, resp)
		requests = append(requests, req)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = db.Get(ctx, requests[i%b.N])
			i++
		}
	})
}

//var BenchmarkWriteIntoStorageNum int
//
//func BenchmarkWriteIntoStorage(b *testing.B) {
//	log.Info().Msg("[" + strconv.Itoa(BenchmarkReadFromStorageNum) + "] Started BenchmarkWriteIntoStorage benchmark with " + strconv.Itoa(b.N) + " iterations.")
//	BenchmarkWriteIntoStorageNum++
//
//	s := New(config.Storage{
//		InitStorageLengthPerShard: 128,
//		EvictionAlgo:              string(algo.LRU),
//		MemoryFillThreshold:       0.95,
//		MemoryLimit:               1024 * 1024 * 128,
//	})
//
//	ctx := context.Background()
//
//	seoRepo := repository.NewSeo(config.Repository{SeoUrl: "https://seo-master.lux.kube.xbet.lan/api/v2/pagedata"})
//
//	cfg := config.Response{
//		RevalidateBeta:     0.5,
//		RevalidateInterval: time.Minute * 10,
//	}
//
//	responses := make([]*model.Response, b.N)
//	for i := 0; i < b.N; i++ {
//		req := model.NewRequest("285", "1xbet.com", "en", `{"name": "betting", "choice": null}`+strconv.Itoa(i))
//		resp, err := model.NewResponse(
//			cfg, http.Header{}, req, 200, []byte(`{"data": "success"}`),
//			func() (statusCode int, body []byte, headers http.Header, err error) {
//				return seoRepo.PageData(ctx, req)
//			},
//		)
//		if err != nil {
//			panic(err)
//		}
//		responses[i] = resp
//	}
//
//	ii := 0
//	tt := time.Duration(0)
//	b.ResetTimer()
//	b.RunParallel(func(pb *testing.PB) {
//		i := 0
//		t := time.Duration(0)
//		for pb.Next() {
//			tc := time.Now()
//			s.Set(ctx, responses[i%b.N])
//			t += time.Since(tc)
//			i++
//		}
//		if i != 0 {
//			log.Info().Msgf("["+strconv.Itoa(BenchmarkWriteIntoStorageNum)+
//				"] BenchmarkWriteIntoStorage b.N: %d, avg duration: %s ns/op", i, strconv.Itoa((int(t.Nanoseconds())/i)/10))
//		}
//		tt += t
//		ii += i
//	})
//	log.Info().Msgf("["+strconv.Itoa(BenchmarkWriteIntoStorageNum)+
//		"] TOTAL --->>> BenchmarkWriteIntoStorage total b.N: %d, total avg duration: %s ns/op", ii, strconv.Itoa((int(tt.Nanoseconds())/ii)/10))
//}
