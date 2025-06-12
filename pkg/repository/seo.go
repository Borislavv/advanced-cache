package repository

import (
	"context"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"net/http"
	"time"
)

const requestTimeout = time.Second * 10

// Seo defines the interface for a repository that provides SEO page data.
type Seo interface {
	PageData(ctx context.Context, req *model.Request) (*model.Response, error)
}

// SeoRepository implements the Seo interface.
// It fetches and constructs SEO page data responses from an external backend.
type SeoRepository struct {
	cfg    *config.Config      // Global configuration (SEO backend URL, etc)
	reader synced.PooledReader // Efficient reader for HTTP responses
}

// NewSeo creates a new instance of SeoRepository.
func NewSeo(cfg *config.Config, reader synced.PooledReader) *SeoRepository {
	return &SeoRepository{
		cfg:    cfg,
		reader: reader,
	}
}

// PageData fetches page data for the given request and constructs a cacheable response.
// It also attaches a revalidator closure for future background refreshes.
func (s *SeoRepository) PageData(ctx context.Context, req *model.Request) (*model.Response, error) {
	// Fetch data from backend.
	data, err := s.requestPagedata(ctx, req)
	if err != nil {
		return nil, errors.New("failed to request pagedata: " + err.Error())
	}

	// Prepare a closure to allow future revalidation using the same logic/endpoint.
	revalidator := func(ctx context.Context) (*model.Data, error) {
		return s.requestPagedata(ctx, req)
	}

	// Build a new response object, which contains the cache payload, request, config and revalidator.
	resp, err := model.NewResponse(data, req, s.cfg, revalidator)
	if err != nil {
		return nil, errors.New("failed to create response: " + err.Error())
	}

	return resp, nil
}

// requestPagedata actually performs the HTTP request to the SEO backend and parses the response.
// Returns a Data object suitable for caching.
func (s *SeoRepository) requestPagedata(ctx context.Context, req *model.Request) (*model.Data, error) {
	// Apply a hard timeout for the HTTP request.
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	url := s.cfg.SeoUrl
	query := req.ToQuery()

	// Efficiently concatenate base URL and query.
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

	// Read response body using a pooled reader to reduce allocations.
	body, freeFn, err := s.reader.Read(response)
	if err != nil {
		return nil, err
	}

	return model.NewData(response.StatusCode, response.Header, body, freeFn), nil
}
