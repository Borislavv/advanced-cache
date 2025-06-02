package storage

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var bts = []byte("{success: true}")

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

var BenchmarkReadFromStorageNum int

func BenchmarkReadFromStorage(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().Msgf("[" + strconv.Itoa(BenchmarkReadFromStorageNum) + "] Started BenchmarkReadFromStorage benchmark with " + strconv.Itoa(b.N) + " iterations.")
	BenchmarkReadFromStorageNum++

	cfg := config.Config{
		Storage: config.Storage{
			InitStorageLengthPerShard: 256,
			EvictionAlgo:              "LRU",
			MemoryFillThreshold:       0.95,
			MemoryLimit:               1024 * 1024 * 12,
		},
		Response: config.Response{
			RevalidateBeta:     0.5,
			RevalidateInterval: time.Minute * 120,
		},
		Repository: config.Repository{
			SeoUrl: "https://seo-master.lux.kube.xbet.lan/api/v2/pagedata",
		},
	}

	db := New(cfg)
	requests := make([]*model.Request, 0, b.N)
	for i := 0; i < b.N; i++ {
		req := model.NewRequest("285", "1xbet.com", "en", `{"name": "betting", "choice": null}`)
		_, _, _ = db.Get(ctx, req, func(ctx context.Context, req *model.Request) (statusCode int, body []byte, headers http.Header, err error) {
			return 200, []byte(`{"data": {"success": true}}`), http.Header{}, nil
		})
		requests = append(requests, req)
	}

	//// Создаём файл для сохранения профиля
	//f, err := os.Create("cpu.pprof")
	//if err != nil {
	//	panic("could not create CPU profile: " + err.Error())
	//}
	//defer f.Close()
	//
	//// Запускаем CPU-профилирование
	//if err = pprof.StartCPUProfile(f); err != nil {
	//	panic("could not start CPU profile: " + err.Error())
	//}
	//defer pprof.StopCPUProfile() // важно остановить!

	//ii := 0
	//tt := time.Duration(0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		//t := time.Duration(0)
		for pb.Next() {
			//tc := time.Now()
			_, _, _ = db.Get(ctx, requests[i%b.N], func(ctx context.Context, req *model.Request) (statusCode int, body []byte, headers http.Header, err error) {
				return 200, []byte(`{"data": {"success": true}}`), http.Header{}, nil
			})
			//if err != nil {
			//	panic(err)
			//}
			//if !hit {
			//	log.Info().Msg("request was not found in storage")
			//}
			//t += time.Since(tc)
			i++
		}
		//if i != 0 {
		//	logger.InfoMsgf(ctx, nil, "["+strconv.Itoa(BenchmarkReadFromStorageNum)+
		//		"] BenchmarkReadFromStorage b.N: %d, avg duration: %s ns/op", i, strconv.Itoa((int(t.Nanoseconds())/i)/10))
		//}
		//tt += t
		//ii += i
	})
	//logger.InfoMsgf(ctx, nil, "["+strconv.Itoa(BenchmarkReadFromStorageNum)+
	//	"] TOTAL --->>> BenchmarkReadFromStorage total b.N: %d, total avg duration: %s ns/op", ii, strconv.Itoa((int(tt.Nanoseconds())/ii)/10))
}

//var BenchmarkWriteIntoStorageNum int
//
//func BenchmarkWriteIntoStorage(b *testing.B) {
//	log.Info().Msg("[" + strconv.Itoa(BenchmarkReadFromStorageNum) + "] Started BenchmarkWriteIntoStorage benchmark with " + strconv.Itoa(b.N) + " iterations.")
//	BenchmarkWriteIntoStorageNum++
//
//	cfg := config.Config{
//		Storage: config.Storage{
//			InitStorageLengthPerShard: 256,
//			EvictionAlgo:              "LRU",
//			MemoryFillThreshold:       0.95,
//			MemoryLimit:               1024 * 1024 * 128,
//		},
//		Response: config.Response{
//			RevalidateBeta:     0.5,
//			RevalidateInterval: time.Minute * 1,
//		},
//		Repository: config.Repository{
//			SeoUrl: "https://seo-master.lux.kube.xbet.lan/api/v2/pagedata",
//		},
//	}
//
//	s := New(cfg)
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	//seoRepo := repository.NewSeo(config.Repository{SeoUrl: "https://seo-master.lux.kube.xbet.lan/api/v2/pagedata"})
//
//	requests := make([]*model.Request, b.N)
//	for i := 0; i < b.N; i++ {
//		req := model.NewRequest("285", "1xbet.com", "en", `{"name": "betting", "choice": null}`+strconv.Itoa(i))
//		requests[i] = req
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
//			s.Get(ctx, requests[i%b.N], func(ctx context.Context, req *model.Request) (statusCode int, body []byte, headers http.Header, err error) {
//				return 200, []byte(`{"data": {"success": true}}`), http.Header{}, nil
//			})
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
