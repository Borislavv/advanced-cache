package storage

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	"net/http"
)

type Storage interface {
	// Get - returns the same slice headers as stored in origin *model.Response! Don't mutate it, just copy if you need some more than read.
	Get(ctx context.Context, req *model.Request, fn model.ResponseCreator) (statusCode int, body []byte, headers http.Header, found bool, err error)
	Del(req *model.Request)
}

type AlgoStorage struct {
	Storage
}

func New(cfg config.Config) *AlgoStorage {
	var s Storage
	switch cache.Algorithm(cfg.EvictionAlgo) {
	case cache.LRU:
		s = cache.NewLRU(cfg)
	default:
		panic("algorithm \"" + cfg.EvictionAlgo + "\" does not implemented yet")
	}
	return &AlgoStorage{Storage: s}
}
