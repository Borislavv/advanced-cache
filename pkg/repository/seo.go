package repository

import (
	"context"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"net/http"
)

type Seo interface {
	PageData(ctx context.Context, req *model.Request) (resp *model.Response, err error)
}

type SeoRepository struct {
	cfg    *config.Config
	reader synced.PooledReader
}

func NewSeo(cfg *config.Config, reader synced.PooledReader) *SeoRepository {
	return &SeoRepository{
		cfg:    cfg,
		reader: reader,
	}
}

func (s *SeoRepository) PageData(ctx context.Context, req *model.Request) (resp *model.Response, err error) {
	data, err := s.requestPagedata(ctx, req)
	if err != nil {
		return nil, errors.New("failed to request pagedata: " + err.Error())
	}

	revalidator := func(ctx context.Context) (data *model.Data, err error) {
		return s.requestPagedata(ctx, req)
	}

	resp, err = model.NewResponse(data, req, s.cfg, revalidator)
	if err != nil {
		return nil, errors.New("failed to create response: " + err.Error())
	}

	return resp, nil
}

func (s *SeoRepository) requestPagedata(ctx context.Context, req *model.Request) (data *model.Data, err error) {
	url := s.cfg.SeoUrl

	query := req.ToQuery()
	queryBuf := make([]byte, 0, len(url)+len(query))
	for _, rn := range url {
		queryBuf = append(queryBuf, byte(rn))
	}
	queryBuf = append(queryBuf, query...)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, string(queryBuf), nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	body, freeFn, err := s.reader.Read(response)
	if err != nil {
		return nil, err
	}

	return model.NewData(response.StatusCode, response.Header, body, freeFn), nil
}
