package lru

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/mock"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	time2 "github.com/Borislavv/traefik-http-cache-plugin/pkg/time"
	"testing"
	"time"
)

var (
	cfg = &config.Config{
		AppEnv:                    "dev",
		AppDebug:                  true,
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       1,
		MemoryLimit:               1024 * 100, // 100kb
	}
	shardedMap = sharded.NewMap[*model.Response](cfg.InitStorageLengthPerShard)
	lru        = NewLRU(context.Background(), cfg, time2.NewWheel(context.Background(), shardedMap), shardedMap)
	responses  = mock.GenerateRandomResponses(cfg, 1000000)
)

func TestSwap(t *testing.T) {
	for _, resp := range responses {
		lru.Set(resp)
	}
	ch := make(chan struct{})
	go func() {
		time.Sleep(time.Second * 2)
		ch <- struct{}{}
	}()
	<-ch
}
