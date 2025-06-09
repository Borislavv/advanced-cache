package storage

import (
	"context"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"testing"
	"time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/mock"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	"github.com/rs/zerolog/log"
)

var BenchmarkNum int

func BenchmarkReadFromStorage1000TimesPerIter(b *testing.B) {
	BenchmarkNum++
	log.Info().Msg("[" + strconv.Itoa(BenchmarkNum) + "] Started Read Benchmark: " + strconv.Itoa(b.N) + " iterations.")

	ctx := context.Background()
	cfg := &config.Config{
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024 * 1024 * 3,
	}
	db := New(ctx, cfg)
	responses := mock.GenerateRandomResponses(cfg, b.N+1)
	for _, resp := range responses {
		db.Set(resp)
	}
	length := len(responses)

	// 🧠 Start profiling
	cpuFile, _ := os.Create("cpu_read.prof")
	defer cpuFile.Close()
	pprof.StartCPUProfile(cpuFile)
	defer pprof.StopCPUProfile()

	memFileBefore, _ := os.Create("mem_before.prof")
	defer memFileBefore.Close()

	memFileAfter, _ := os.Create("mem_after.prof")
	defer memFileAfter.Close()

	traceFile, _ := os.Create("trace_read.out")
	defer traceFile.Close()
	trace.Start(traceFile)
	defer trace.Stop()

	runtime.GC()
	pprof.WriteHeapProfile(memFileBefore)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			for j := 0; j < 1000; j++ {
				_, release, _ := db.Get(responses[(i*j)%length].GetRequest())
				release()
			}
			i += 1000
		}
	})
	b.StopTimer()

	runtime.GC()
	pprof.WriteHeapProfile(memFileAfter)

	log.Info().Msg("[" + strconv.Itoa(BenchmarkNum) + "] Read Benchmark done.")
}

func BenchmarkWriteIntoStorage1000TimesPerIter(b *testing.B) {
	BenchmarkNum++
	log.Info().Msg("[" + strconv.Itoa(BenchmarkNum) + "] Started Write Benchmark: " + strconv.Itoa(b.N) + " iterations.")

	ctx := context.Background()
	cfg := &config.Config{
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024 * 1024 * 3,
	}
	db := New(ctx, cfg)
	responses := mock.GenerateRandomResponses(cfg, b.N+1)
	length := len(responses)

	cpuFile, _ := os.Create("cpu_write.prof")
	defer cpuFile.Close()
	pprof.StartCPUProfile(cpuFile)
	defer pprof.StopCPUProfile()

	memFileBefore, _ := os.Create("mem_before.prof")
	defer memFileBefore.Close()

	memFileAfter, _ := os.Create("mem_after.prof")
	defer memFileAfter.Close()

	traceFile, _ := os.Create("trace_write.out")
	defer traceFile.Close()
	trace.Start(traceFile)
	defer trace.Stop()

	runtime.GC()
	pprof.WriteHeapProfile(memFileBefore)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			for j := 0; j < 1000; j++ {
				db.Get(responses[(i*j)%length].GetRequest())
			}
			i += 100
		}
	})
	b.StopTimer()

	runtime.GC()
	pprof.WriteHeapProfile(memFileAfter)

	log.Info().Msg("[" + strconv.Itoa(BenchmarkNum) + "] Write Benchmark done.")
}

func BenchmarkGetAllocs(b *testing.B) {
	ctx := context.Background()
	cfg := &config.Config{
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024,
	}
	db := New(ctx, cfg)
	resp := mock.GenerateRandomResponses(cfg, 1)[0]
	db.Set(resp)
	req := resp.GetRequest()

	allocs := testing.AllocsPerRun(100000, func() {
		db.Get(req)
	})
	b.ReportMetric(allocs, "allocs/op")
}

func BenchmarkSetAllocs(b *testing.B) {
	ctx := context.Background()
	cfg := &config.Config{
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024,
	}
	db := New(ctx, cfg)
	resp := mock.GenerateRandomResponses(cfg, 1)[0]

	allocs := testing.AllocsPerRun(100000, func() {
		db.Set(resp)
	})
	b.ReportMetric(allocs, "allocs/op")
}
