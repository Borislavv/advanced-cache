package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
)

type Seo interface {
	PageData(ctx context.Context, req *model.Request) (resp *model.Response, err error)
}

type SeoRepository struct {
	cfg *config.Config
}

func NewSeo(cfg *config.Config) *SeoRepository {
	return &SeoRepository{
		cfg: cfg,
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

	if response.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(response.Body)
		fmt.Println(string(b), string(queryBuf))
		return nil, errors.New("non-positive " + strconv.Itoa(response.StatusCode) + " status code received from pagedata")
	}

	body, err := utils.ReadResponseBody(response)
	if err != nil {
		return nil, err
	}

	return model.NewData(response.StatusCode, response.Header, body), nil
}
