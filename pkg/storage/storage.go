package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache/lru"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
)

type Storage interface {
	Get(req *model.Request) (resp *model.Response, found bool)
	Set(resp *model.Response)
	Del(req *model.Request)
}

type AlgoStorage struct {
	Storage
}

func New(ctx context.Context, cfg *config.Config) *AlgoStorage {
	var s Storage
	switch cache.Algorithm(cfg.EvictionAlgo) {
	case cache.LRU:
		s = lru.NewLRU(ctx, cfg, lru.NewBalancer())
	default:
		panic("algorithm " + cfg.EvictionAlgo + " does not implemented yet")
	}
	return &AlgoStorage{Storage: s}
}
