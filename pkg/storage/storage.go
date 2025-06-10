package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache/lru"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/wheel"
	wheelmodel "github.com/Borislavv/traefik-http-cache-plugin/pkg/wheel/model"
)

type Storage interface {
	Get(req *model.Request) (resp *model.Response, releaser *sharded.Releaser[*model.Response], isHit bool)
	Set(resp *model.Response) (releaser *sharded.Releaser[*model.Response])
}

type AlgoStorage struct {
	Storage
}

func New(ctx context.Context, cfg *config.Config, timeWheel *wheel.OfTime[wheelmodel.Spoke], shardedMap *sharded.Map[*model.Response]) *AlgoStorage {
	var s Storage
	switch cache.Algorithm(cfg.EvictionAlgo) {
	case cache.LRU:
		s = lru.NewLRU(ctx, cfg, timeWheel, shardedMap)
	default:
		panic("algorithm " + cfg.EvictionAlgo + " does not implemented yet")
	}
	return &AlgoStorage{Storage: s}
}
