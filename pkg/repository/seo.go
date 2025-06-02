package repository

import (
	"context"
	"errors"
	"net/http"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
)

type Seo interface {
	PageData(ctx context.Context, req *model.Request) (statusCode int, body []byte, headers http.Header, err error)
}

type SeoRepository struct {
	cfg config.Repository
}

func NewSeo(cfg config.Repository) *SeoRepository {
	return &SeoRepository{
		cfg: cfg,
	}
}

func (s *SeoRepository) PageData(ctx context.Context, req *model.Request) (statusCode int, body []byte, headers http.Header, err error) {
	query, err := req.ToQuery()
	if err != nil {
		// log.Err(err).Msg("failed to build query")
		return 0, nil, nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.SeoUrl+query, nil)
	if err != nil {
		// log.Err(err).Msg("failed to build request")
		return 0, nil, nil, err
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		// log.Err(err).Msg("failed to fetch pagedata")
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil, nil, errors.New("not 200 status code received from pagedata")
	}

	body, err = utils.ReadResponseBody(resp)
	if err != nil {
		// log.Err(err).Msg("failed to read response body")
		return resp.StatusCode, nil, nil, err
	}

	return resp.StatusCode, body, resp.Header, nil
}
