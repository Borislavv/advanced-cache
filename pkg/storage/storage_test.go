package storage

import (
	"context"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"testing"
	"time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/mock"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
)

// BenchmarkReadFromStorage1000TimesPerIter benchmarks parallel cache reads with profile collection.
// Each iteration does 1000 Get() calls with different requests.
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
		MemoryLimit:               1024 * 1024 * 1024 * 3, // 3GB
	}
	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	db := New(ctx, cfg, shardedMap)
	responses := mock.GenerateRandomResponses(cfg, b.N+1)
	for _, resp := range responses {
		db.Set(resp)
	}
	length := len(responses)

	cpuFile, err := os.Create("cpu_read.prof")
	if err != nil {
		panic("failed to create cpu_read.prof: " + err.Error())
	}
	defer cpuFile.Close()
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		panic("failed to start CPU profile: " + err.Error())
	}
	defer pprof.StopCPUProfile()

	memFileBefore, err := os.Create("mem_before.prof")
	if err != nil {
		panic("failed to create mem_before.prof: " + err.Error())
	}
	defer memFileBefore.Close()

	memFileAfter, err := os.Create("mem_after.prof")
	if err != nil {
		panic("failed to create mem_after.prof: " + err.Error())
	}
	defer memFileAfter.Close()

	traceFile, err := os.Create("trace_read.out")
	if err != nil {
		panic("failed to create trace_read.out: " + err.Error())
	}
	defer traceFile.Close()
	if err := trace.Start(traceFile); err != nil {
		panic("failed to start trace: " + err.Error())
	}
	defer trace.Stop()

	runtime.GC()
	if err := pprof.WriteHeapProfile(memFileBefore); err != nil {
		panic("failed to write heap profile (before): " + err.Error())
	}

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
	if err := pprof.WriteHeapProfile(memFileAfter); err != nil {
		panic("failed to write heap profile (after): " + err.Error())
	}
}

// BenchmarkWriteIntoStorage1000TimesPerIter benchmarks parallel cache writes with profile collection.
// Each iteration does 1000 Set() calls.
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
		MemoryLimit:               1024 * 1024 * 1024 * 3, // 3GB
	}
	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	db := New(ctx, cfg, shardedMap)
	responses := mock.GenerateRandomResponses(cfg, b.N+1)
	length := len(responses)

	cpuFile, err := os.Create("cpu_write.prof")
	if err != nil {
		panic("failed to create cpu_write.prof: " + err.Error())
	}
	defer cpuFile.Close()
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		panic("failed to start CPU profile: " + err.Error())
	}
	defer pprof.StopCPUProfile()

	memFileBefore, err := os.Create("mem_before.prof")
	if err != nil {
		panic("failed to create mem_before.prof: " + err.Error())
	}
	defer memFileBefore.Close()

	memFileAfter, err := os.Create("mem_after.prof")
	if err != nil {
		panic("failed to create mem_after.prof: " + err.Error())
	}
	defer memFileAfter.Close()

	traceFile, err := os.Create("trace_write.out")
	if err != nil {
		panic("failed to create trace_write.out: " + err.Error())
	}
	defer traceFile.Close()
	if err := trace.Start(traceFile); err != nil {
		panic("failed to start trace: " + err.Error())
	}
	defer trace.Stop()

	runtime.GC()
	if err := pprof.WriteHeapProfile(memFileBefore); err != nil {
		panic("failed to write heap profile (before): " + err.Error())
	}

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
	if err := pprof.WriteHeapProfile(memFileAfter); err != nil {
		panic("failed to write heap profile (after): " + err.Error())
	}
}

// BenchmarkGetAllocs benchmarks allocation count per Get() call.
func BenchmarkGetAllocs(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024, // 1MB
	}

	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	db := New(ctx, cfg, shardedMap)
	resp := mock.GenerateRandomResponses(cfg, 1)[0]
	db.Set(resp)
	req := resp.Request()

	allocs := testing.AllocsPerRun(100_000, func() {
		db.Get(req)
	})
	b.ReportMetric(allocs, "allocs/op")
}

// BenchmarkSetAllocs benchmarks allocation count per Set() call.
func BenchmarkSetAllocs(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.95,
		MemoryLimit:               1024 * 1024, // 1MB
	}

	shardedMap := sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	db := New(ctx, cfg, shardedMap)
	resp := mock.GenerateRandomResponses(cfg, 1)[0]

	allocs := testing.AllocsPerRun(100_000, func() {
		db.Set(resp)
	})
	b.ReportMetric(allocs, "allocs/op")
}
