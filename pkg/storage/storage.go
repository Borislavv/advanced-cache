package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
)

type Storage interface {
	Get(ctx context.Context, req *model.Request) (resp *model.Response, found bool)
	Set(ctx context.Context, resp *model.Response)
	Del(req *model.Request)
}

type AlgoStorage struct {
	Storage
}

func New(ctx context.Context, cfg *config.Config) *AlgoStorage {
	var s Storage
	switch cache.Algorithm(cfg.EvictionAlgo) {
	case cache.LRU:
		s = cache.NewLRU(ctx, cfg)
	default:
		panic(cfg.EvictionAlgo + " algorithm does not implemented yet")
	}
	return &AlgoStorage{Storage: s}
}
