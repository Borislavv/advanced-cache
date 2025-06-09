package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache/lru"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
)

type Storage interface {
	Get(req *model.Request) (resp *model.Response, releaser *sharded.Releaser[*model.Response], isHit bool)
	Set(resp *model.Response) (releaser *sharded.Releaser[*model.Response])
}

type AlgoStorage struct {
	Storage
}

func New(ctx context.Context, cfg *config.Config, shardedMap *sharded.Map[*model.Response]) *AlgoStorage {
	var s Storage
	switch cache.Algorithm(cfg.EvictionAlgo) {
	case cache.LRU:
		s = lru.NewLRU(ctx, cfg, shardedMap)
	default:
		panic("algorithm " + cfg.EvictionAlgo + " does not implemented yet")
	}
	return &AlgoStorage{Storage: s}
}
