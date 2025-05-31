package service

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
)

type Proxy struct {
	storage storage.Storage
	Refresher
}

func NewProxy(storage storage.Storage, refresher Refresher) *Proxy {
	return &Proxy{
		storage:   storage,
		Refresher: refresher,
	}
}

func (p *Proxy) Get(ctx context.Context, req *model.Request) (data []byte, isFound bool) {
	resp, found := p.storage.Get(req)
	if !found {
		return nil, false
	}
	if p.IsTimeForRefresh(resp) {
		p.Refresh(ctx, resp)
	} else {
		p.Touch(ctx, resp)
	}
	return resp.GetData(), true
}
