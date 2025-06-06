package lru

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/mock"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
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
	lru       = NewLRU(context.Background(), cfg)
	responses = mock.GenerateRandomResponses(cfg, 1000000)
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
