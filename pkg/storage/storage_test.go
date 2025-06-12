package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"testing"
	"time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/mock"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
)

func BenchmarkReadFromStorage1000TimesPerIter(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024 * 1024 * 3,
	}
	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	db := New(ctx, cfg, shardedMap)
	responses := mock.GenerateRandomResponses(cfg, b.N+1)
	for _, resp := range responses {
		db.Set(resp)
	}
	length := len(responses)

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
				_, release, _ := db.Get(responses[(i*j)%length].Request())
				release.Release()
			}
			i += 1000
		}
	})
	b.StopTimer()

	runtime.GC()
	pprof.WriteHeapProfile(memFileAfter)
}

func BenchmarkWriteIntoStorage1000TimesPerIter(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024 * 1024 * 3,
	}
	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	db := New(ctx, cfg, shardedMap)
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
				release := db.Set(responses[(i*j)%length])
				release.Release()
			}
			i += 100
		}
	})
	b.StopTimer()

	runtime.GC()
	pprof.WriteHeapProfile(memFileAfter)
}

func BenchmarkGetAllocs(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := &config.Config{
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024,
	}

	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	db := New(ctx, cfg, shardedMap)
	resp := mock.GenerateRandomResponses(cfg, 1)[0]
	db.Set(resp)
	req := resp.Request()

	allocs := testing.AllocsPerRun(100000, func() {
		db.Get(req)
	})
	b.ReportMetric(allocs, "allocs/op")

	cancel()
}

func BenchmarkSetAllocs(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := &config.Config{
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024,
	}

	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	db := New(ctx, cfg, shardedMap)
	resp := mock.GenerateRandomResponses(cfg, 1)[0]

	allocs := testing.AllocsPerRun(100000, func() {
		db.Set(resp)
	})
	b.ReportMetric(allocs, "allocs/op")

	cancel()
}
