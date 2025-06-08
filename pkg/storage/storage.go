package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache/lru"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
)

type Storage interface {
	Get(req *model.Request) (resp *model.Response, acquire func(), isHit bool)
	Set(resp *model.Response)
}

type AlgoStorage struct {
	Storage
}

func New(ctx context.Context, cfg *config.Config, wheel *time.Wheel, shardedMap *sharded.Map[*model.Response]) *AlgoStorage {
	var s Storage
	switch cache.Algorithm(cfg.EvictionAlgo) {
	case cache.LRU:
		s = lru.NewLRU(ctx, cfg, wheel, shardedMap)
	default:
		panic("algorithm " + cfg.EvictionAlgo + " does not implemented yet")
	}
	return &AlgoStorage{Storage: s}
}
