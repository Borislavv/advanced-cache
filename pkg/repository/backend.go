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

// Backender defines the interface for a repository that provides SEO page data.
type Backender interface {
	Fetch(ctx context.Context, req *model.Request) (*model.Response, error)
	RevalidatorMaker(req *model.Request) func(ctx context.Context) (*model.Data, error)
}

// Backend implements the Backender interface.
// It fetches and constructs SEO page data responses from an external backend.
type Backend struct {
	cfg    *config.Cache       // Global configuration (SEO backend URL, etc)
	reader synced.PooledReader // Efficient reader for HTTP responses
}

// NewBackend creates a new instance of Backend.
func NewBackend(cfg *config.Cache, reader synced.PooledReader) *Backend {
	return &Backend{
		cfg:    cfg,
		reader: reader,
	}
}

// Fetch method fetches page data for the given request and constructs a cacheable response.
// It also attaches a revalidator closure for future background refreshes.
func (s *Backend) Fetch(ctx context.Context, req *model.Request) (*model.Response, error) {
	// Fetch data from backend.
	data, err := s.requestExternalBackend(ctx, req)
	if err != nil {
		return nil, errors.New("failed to request external backend: " + err.Error())
	}

	// Prepare a closure to allow future revalidation using the same logic/endpoint.
	Revalidator := func(ctx context.Context) (*model.Data, error) {
		return s.requestExternalBackend(ctx, req)
	}

	// Build a new response object, which contains the cache payload, request, config and revalidator.
	resp, err := model.NewResponse(data, req, s.cfg, Revalidator)
	if err != nil {
		return nil, errors.New("failed to create response: " + err.Error())
	}

	return resp, nil
}

func (s *Backend) RevalidatorMaker(req *model.Request) func(ctx context.Context) (*model.Data, error) {
	return func(ctx context.Context) (*model.Data, error) {
		return s.requestExternalBackend(ctx, req)
	}
}

// requestExternalBackend actually performs the HTTP request to backend and parses the response.
// Returns a Data object suitable for caching.
func (s *Backend) requestExternalBackend(ctx context.Context, req *model.Request) (*model.Data, error) {
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
