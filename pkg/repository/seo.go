package repository

import (
	"context"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"net/http"
	"strconv"
)

type Seo interface {
	PageData(ctx context.Context, req *model.Request) (statusCode int, body []byte, headers http.Header, err error)
}

type SeoRepository struct {
	cfg *config.Config
}

func NewSeo(cfg *config.Config) *SeoRepository {
	return &SeoRepository{
		cfg: cfg,
	}
}

func (s *SeoRepository) PageData(ctx context.Context, req *model.Request) (statusCode int, body []byte, headers http.Header, err error) {
	query, err := req.ToQuery()
	if err != nil {
		return http.StatusServiceUnavailable, nil, nil, errors.New("failed to build query: " + err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.SeoUrl+query, nil)
	if err != nil {
		return http.StatusServiceUnavailable, nil, nil, errors.New("failed to build request: " + err.Error())
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return http.StatusServiceUnavailable, nil, nil, errors.New("failed to fetch pagedata: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil, nil,
			errors.New("not 200 status code (actual: " + strconv.Itoa(resp.StatusCode) + ") received from pagedata")
	}

	body, err = utils.ReadResponseBody(resp)
	if err != nil {
		return resp.StatusCode, nil, nil, errors.New("failed to read response body: " + err.Error())
	}

	return resp.StatusCode, body, resp.Header, nil
}
