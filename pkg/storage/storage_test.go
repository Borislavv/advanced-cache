package storage

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/mock"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
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

var BenchmarkReadFromStorageNum int

func BenchmarkReadFromStorage1000TimesPerIter(b *testing.B) {
	log.Info().Msg("[" + strconv.Itoa(BenchmarkReadFromStorageNum) +
		"] Started BenchmarkReadFromStorage benchmark with " + strconv.Itoa(b.N) + " iterations.")
	BenchmarkReadFromStorageNum++

	ctx := context.Background()

	cfg := &config.Config{
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024 * 1024 * 3, // 3GB
	}

	db := New(ctx, cfg)

	responses := mock.GenerateRandomResponses(cfg, b.N+1)
	for _, resp := range responses {
		db.Set(ctx, resp)
	}
	length := len(responses)

	log.Info().Msg("[" + strconv.Itoa(BenchmarkReadFromStorageNum) +
		"] BenchmarkReadFromStorage benchmark generated " + strconv.Itoa(b.N) + " mock items.")

	from := time.Now()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			for j := 0; j < 1000; j++ {
				_, _ = db.Get(ctx, responses[(i+j)%length].GetRequest())
			}
			i += 1000
		}
	})
	b.StopTimer()

	log.Info().Msg("[" + strconv.Itoa(BenchmarkReadFromStorageNum) +
		"] BenchmarkReadFromStorage benchmark " + strconv.Itoa(b.N) + " iterations elapsed time: " + time.Since(from).String() + ".")
}

var BenchmarkWriteIntoStorageNum int

func BenchmarkWriteIntoStorage1000TimesPerIter(b *testing.B) {
	log.Info().Msg("[" + strconv.Itoa(BenchmarkReadFromStorageNum) +
		"] Started BenchmarkWriteIntoStorage benchmark with " + strconv.Itoa(b.N) + " iterations.")
	BenchmarkWriteIntoStorageNum++

	ctx := context.Background()

	cfg := &config.Config{
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024 * 1024 * 3, // 3GB
	}

	db := New(ctx, cfg)

	responses := mock.GenerateRandomResponses(cfg, b.N+1)
	length := len(responses)

	log.Info().Msg("[" + strconv.Itoa(BenchmarkReadFromStorageNum) +
		"] BenchmarkWriteIntoStorage benchmark generated " + strconv.Itoa(b.N) + " mock items.")

	from := time.Now()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			for j := 0; j < 1000; j++ {
				resp := responses[(i+j)%length]
				db.Set(ctx, resp)
			}
			i += 1000
		}
	})
	b.StopTimer()

	log.Info().Msg("[" + strconv.Itoa(BenchmarkWriteIntoStorageNum) +
		"] BenchmarkWriteIntoStorage benchmark " + strconv.Itoa(b.N) + " iterations elapsed time: " + time.Since(from).String() + ".")
}
