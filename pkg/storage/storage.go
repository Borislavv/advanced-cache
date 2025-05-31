package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/algo"
)

type Storage interface {
	Get(req *model.Request) (resp *model.Response, found bool)
	Set(ctx context.Context, resp *model.Response)
	Del(req *model.Request)
}

type AlgoStorage struct {
	Storage
}

func New(cfg config.Storage, defaultLen int) *AlgoStorage {
	var s Storage
	switch algo.Algorithm(cfg.EvictionAlgo) {
	case algo.LRU:
		s = algo.NewLRU(cfg, defaultLen)
	default:
		panic(cfg.EvictionAlgo + " algorithm does not implemented yet")
	}
	return &AlgoStorage{Storage: s}
}
