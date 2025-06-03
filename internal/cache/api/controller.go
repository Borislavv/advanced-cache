package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	"github.com/fasthttp/router"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	util "gitlab.xbet.lan/v3group/backend/packages/go/httpserver/pkg/httpserver/util"
	"net/http"
	"time"
)

const CacheGetPath = "/api/v1/cache/pagedata"

var (
	badRequestResponseBytes = []byte(`{
	  "status": 400,
	  "error": "Bad Request",
	  "message": "` + string(messagePlaceholder) + `"
	}`)
	serviceUnavailableResponseBytes = []byte(`{
	  "status": 503,
	  "error": "Service Unavailable",
	  "message": "` + string(messagePlaceholder) + `",
	}`)
	messagePlaceholder = []byte("${message}")
)

type CacheController struct {
	cfg     *config.Config
	ctx     context.Context
	cache   storage.Storage
	seoRepo repository.Seo
}

func NewCacheController(
	ctx context.Context,
	cfg *config.Config,
	cache storage.Storage,
	seoRepo repository.Seo,
) *CacheController {
	return &CacheController{
		cfg:     cfg,
		ctx:     ctx,
		cache:   cache,
		seoRepo: seoRepo,
	}
}

func (c *CacheController) Index(r *fasthttp.RequestCtx) {
	from := time.Now()

	ctx, err := util.ExtractCtx(r)
	if err != nil {
		ctx = c.ctx
		log.Warn().Msg(err.Error())
	}

	req, err := c.makeModelRequest(r)
	if err != nil {
		c.respondThatTheRequestIsBad(err, r)
		return
	}

	resp, found := c.cache.Get(ctx, req)
	if !found {
		resp, err = c.seoRepo.PageData(ctx, req)
		if err != nil {
			c.respondThatServiceIsTemporaryUnavailable(err, r)
			return
		}
		c.cache.Set(ctx, resp)
	}

	data := resp.GetData()
	r.Response.SetStatusCode(data.StatusCode())
	for key, vv := range data.Headers() {
		for _, value := range vv {
			r.Request.Header.Add(key, value)
		}
	}

	r.Response.Header.Add("Last-Modified", resp.GetRevalidatedAt().Format(http.TimeFormat))
	if _, err = util.Write(data.Body(), r); err != nil {
		c.respondThatServiceIsTemporaryUnavailable(err, r)
		return
	}

	log.Debug().Msg("request has been processed in " + time.Since(from).String())
}

func (c *CacheController) respondThatServiceIsTemporaryUnavailable(err error, ctx *fasthttp.RequestCtx) {
	log.Err(err).Msg("error occurred while processing request")

	ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
	if _, err = util.Write(c.resolveMessagePlaceholder(serviceUnavailableResponseBytes, err), ctx); err != nil {
		log.Err(err).Msg("failed to write into *fasthttp.RequestCtx")
	}
}

func (c *CacheController) respondThatTheRequestIsBad(err error, ctx *fasthttp.RequestCtx) {
	log.Err(err).Msg("bad request was caught")

	ctx.SetStatusCode(fasthttp.StatusBadRequest)
	if _, err = util.Write(c.resolveMessagePlaceholder(badRequestResponseBytes, err), ctx); err != nil {
		log.Err(err).Msg("failed to write into *fasthttp.RequestCtx")
	}
}

func (c *CacheController) resolveMessagePlaceholder(msg []byte, err error) []byte {
	escaped, _ := json.Marshal(err.Error())
	return bytes.ReplaceAll(msg, messagePlaceholder, escaped[1:len(escaped)-1])
}

func (c *CacheController) makeModelRequest(r *fasthttp.RequestCtx) (*model.Request, error) {
	project := r.QueryArgs().Peek("project[id]")
	if project == nil || len(project) == 0 {
		return nil, errors.New("project is not specified")
	}
	domain := r.QueryArgs().Peek("domain")
	if domain == nil || len(domain) == 0 {
		return nil, errors.New("domain is not specified")
	}
	language := r.QueryArgs().Peek("language")
	if language == nil || len(language) == 0 {
		return nil, errors.New("language is not specified")
	}
	return model.NewRequest(project, domain, language, model.ExtractTags(r.QueryArgs())), nil
}

func (c *CacheController) AddRoute(router *router.Router) {
	router.GET(CacheGetPath, c.Index)
}
