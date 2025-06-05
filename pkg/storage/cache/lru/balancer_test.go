package lru

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/mock"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	"github.com/rs/zerolog/log"
	"testing"
	"time"
)

var (
	cfg = &config.Config{
		SeoUrl:                    "",
		RevalidateBeta:            0.3,
		RevalidateInterval:        time.Hour,
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       1,
		MemoryLimit:               1024 * 100, // 100kb
	}
	balancer  = NewBalancer()
	lru       = NewLRU(context.Background(), cfg, balancer)
	responses = mock.GenerateRandomResponses(cfg, 1000)
)

func TestSwap(t *testing.T) {
	for _, resp := range responses {
		lru.Set(resp)
	}

	cur := balancer.memList.Front()
	for cur != nil {
		s := cur.Value.(*node).shard
		log.Info().Msgf("shard id: %d, shard len: %d, shard mem: %d", s.ID(), s.Len.Load(), s.Size())
		cur = cur.Next()
	}
}
