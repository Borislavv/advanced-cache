package service

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type Refresher interface {
	Refresh(ctx context.Context, resp *model.Response)
	Touch(ctx context.Context, resp *model.Response)
	IsTimeForRefresh(resp *model.Response) bool
}

type ResponseRefresher struct {
	cfg               config.Refresher
	cluster           storage.Storage
	repository        repository.Seo
	refreshDataCh     chan *model.Response
	refreshMetadataCh chan *model.Response
}

func NewRefresher(ctx context.Context, cfg config.Refresher, cluster storage.Storage, repository repository.Seo) *ResponseRefresher {
	refresher := &ResponseRefresher{
		cfg:               cfg,
		cluster:           cluster,
		repository:        repository,
		refreshDataCh:     make(chan *model.Response, cfg.RefreshParallelism),
		refreshMetadataCh: make(chan *model.Response, cfg.RefreshParallelism),
	}
	refresher.consumeDataRefresh(ctx)
	refresher.consumeMetadataRefresh(ctx)
	return refresher
}

func (r *ResponseRefresher) IsTimeForRefresh(resp *model.Response) bool {
	return resp.GetRevalidatedAt().Add(resp.GetRevalidateInterval()).Unix() <= time.Now().Unix()
}

func (r *ResponseRefresher) Refresh(ctx context.Context, resp *model.Response) {
	select {
	case <-ctx.Done():
	case r.refreshDataCh <- resp:
	}
}

func (r *ResponseRefresher) Touch(ctx context.Context, resp *model.Response) {
	select {
	case <-ctx.Done():
	case r.refreshMetadataCh <- resp:
	}
}

func (r *ResponseRefresher) consumeDataRefresh(ctx context.Context) {
	semaphore := make(chan struct{}, r.cfg.RefreshParallelism)
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case dto := <-r.refreshDataCh:
			semaphore <- struct{}{}
			wg.Add(1)
			go func() {
				defer func() {
					<-semaphore
					wg.Done()
				}()
				r.setData(ctx, dto)
			}()
		}
	}
}

func (r *ResponseRefresher) consumeMetadataRefresh(ctx context.Context) {
	semaphore := make(chan struct{}, r.cfg.RefreshParallelism)
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case dto := <-r.refreshMetadataCh:
			semaphore <- struct{}{}
			wg.Add(1)
			go func() {
				defer func() {
					<-semaphore
					wg.Done()
				}()
				r.setMetadata(ctx, dto)
			}()
		}
	}
}

func (r *ResponseRefresher) setData(ctx context.Context, resp *model.Response) {
	b, err := r.repository.PageData()
	if err != nil {
		log.Error().Err(err).Msg("failed to refresh response due to pagedata is unavailable")
		return
	}

	resp.SetData(b)

	r.cluster.Set(ctx, resp)

	log.Debug().Msg("data successfully updated")
}

func (r *ResponseRefresher) setMetadata(ctx context.Context, resp *model.Response) {
	resp.Touch()

	r.cluster.Set(ctx, resp)

	log.Debug().Msg("metadata successfully updated")
}
