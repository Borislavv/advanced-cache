package api

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Borislavv/traefik-http-cache-plugin/internal/cache/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
	"github.com/fasthttp/router"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	util "gitlab.xbet.lan/v3group/backend/packages/go/httpserver/pkg/httpserver/util"
	"net/http"
	"runtime"
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
	zeroLiteral        = "0"
)

var (
	durCh chan time.Duration
)

type CacheController struct {
	cfg     *config.Config
	ctx     context.Context
	cache   storage.Storage
	seoRepo repository.Seo
	reader  synced.PooledReader
}

func NewCacheController(
	ctx context.Context,
	cfg *config.Config,
	cache storage.Storage,
	seoRepo repository.Seo,
	reader synced.PooledReader,
) *CacheController {
	c := &CacheController{
		cfg:     cfg,
		ctx:     ctx,
		cache:   cache,
		seoRepo: seoRepo,
		reader:  reader,
	}
	if c.cfg.IsDebugOn() {
		c.runLogDebugInfo(ctx)
	}
	return c
}

func (c *CacheController) Index(r *fasthttp.RequestCtx) {
	f := time.Now()

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	req, err := model.NewRequest(r.QueryArgs())
	if err != nil {
		c.respondThatTheRequestIsBad(err, r)
		return
	}

	resp, found := c.cache.Get(req)
	if !found {
		resp, err = c.seoRepo.PageData(ctx, req)
		if err != nil {
			c.respondThatServiceIsTemporaryUnavailable(err, r)
			return
		}
		c.cache.Set(resp)
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

	if c.cfg.IsDebugOn() {
		select {
		case durCh <- time.Since(f):
		default:
		}
	}
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

func (c *CacheController) AddRoute(router *router.Router) {
	router.GET(CacheGetPath, c.Index)
}

type stat struct {
	label    string
	divider  int // in seconds
	tickerCh <-chan time.Time
	count    int
	total    time.Duration
}

func (c *CacheController) runLogDebugInfo(ctx context.Context) {
	durCh = make(chan time.Duration, runtime.GOMAXPROCS(0))

	go func() {
		stats := []*stat{
			{label: "5s", divider: 5, tickerCh: utils.NewTicker(ctx, 5*time.Second)},
			{label: "1m", divider: 60, tickerCh: utils.NewTicker(ctx, time.Minute)},
			{label: "5m", divider: 300, tickerCh: utils.NewTicker(ctx, 5*time.Minute)},
			{label: "1h", divider: 3600, tickerCh: utils.NewTicker(ctx, time.Hour)},
		}

		for {
			select {
			case <-ctx.Done():
				return
			case dur := <-durCh:
				for _, s := range stats {
					s.count++
					s.total += dur
				}
			case <-stats[0].tickerCh:
				c.logAndReset(stats[0])
			case <-stats[1].tickerCh:
				c.logAndReset(stats[1])
			case <-stats[2].tickerCh:
				c.logAndReset(stats[2])
			case <-stats[3].tickerCh:
				c.logAndReset(stats[3])
			}
		}
	}()
}

func (c *CacheController) logAndReset(s *stat) {
	var avg string
	if s.count > 0 {
		avg = (s.total / time.Duration(s.count)).String()
	} else {
		avg = zeroLiteral
	}
	log.Info().Msgf(
		"[stat] RPS: %d, total req: %d (%s), avg duration %s",
		s.count/s.divider, s.count, s.label, avg,
	)
	s.count = 0
	s.total = 0
}
