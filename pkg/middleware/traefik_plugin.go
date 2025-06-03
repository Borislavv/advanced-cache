package middleware

import (
	"bytes"
	"context"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/cache"
	"net/http"
	"time"

	cfg "github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/utils"
)

var internalServerErrorJson = []byte(`{"error": {"message": "Internal server error."}}`)

type Config struct {
	SeoUrl string `mapstructure:"SEO_URL"`
	// RevalidateBeta is a value which will be used for generate
	// random time point for refresh response (must be from 0.1 to 0.9).
	// Don't use absolute values like 0 and 1 due it will be leading to CPU peaks usage.
	//  - beta = 0.5 — regular, good for start value
	//  - beta = 1.0 — aggressive refreshing
	//  - beta = 0.0 — disables refreshing
	RevalidateBeta            float64       `mapstructure:"REVALIDATE_BETA"`
	RevalidateInterval        time.Duration `mapstructure:"REVALIDATE_INTERVAL"`
	InitStorageLengthPerShard int           `mapstructure:"INIT_STORAGE_LEN_PER_SHARD"`
	EvictionAlgo              string        `mapstructure:"EVICTION_ALGO"`
	MemoryFillThreshold       float64       `mapstructure:"MEMORY_FILL_THRESHOLD"`
	MemoryLimit               float64       `mapstructure:"MEMORY_LIMIT"`
}

func CreateConfig() *Config {
	return &Config{
		InitStorageLengthPerShard: 256,
		EvictionAlgo:              string(cache.LRU),
		MemoryFillThreshold:       0.97,
		MemoryLimit:               1024 * 1024 * 256,
		RevalidateBeta:            0.4,
		RevalidateInterval:        time.Minute * 30,
		SeoUrl:                    "https://seo-master.lux.kube.xbet.lan/api/v2/pagedata",
	}
}

type Plugin struct {
	ctx     context.Context
	next    http.Handler
	name    string
	config  *Config
	seoRepo repository.Seo
	storage storage.Storage
}

func New(ctx context.Context, next http.Handler, config *Config, name string) http.Handler {
	return &Plugin{
		ctx:     ctx,
		next:    next,
		name:    name,
		config:  config,
		seoRepo: repository.NewSeo(config),
		storage: storage.New(ctx, cfg.Config{
			SeoUrl:                    config.SeoUrl,
			RevalidateBeta:            config.RevalidateBeta,
			RevalidateInterval:        config.RevalidateInterval,
			InitStorageLengthPerShard: config.InitStorageLengthPerShard,
			EvictionAlgo:              config.EvictionAlgo,
			MemoryFillThreshold:       config.MemoryFillThreshold,
			MemoryLimit:               config.MemoryLimit,
		}),
	}
}

func (p *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.next == nil {
		return
	}

	if w == nil {
		return
	}

	ctx, cancel := context.WithTimeout(p.ctx, time.Second*3)
	defer cancel()

	req, err := p.extractRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if _, werr := w.Write(internalServerErrorJson); werr != nil {
			// log.Err(err).Msg("error while writing response into http.ResponseWriter")
		}
		return
	}

	creator := func(ctx context.Context, req *model.Request) (int, []byte, http.Header, error) {
		clonedWriter := newCaptureResponseWriter(w)
		p.next.ServeHTTP(clonedWriter, r)
		return clonedWriter.statusCode, clonedWriter.body.Bytes(), clonedWriter.Header(), nil
	}

	resp, isHit, err := p.storage.Get(ctx, req, creator)
	if err != nil {
		// log.Err(err).Msg("error while fetching from storage")
		w.WriteHeader(http.StatusInternalServerError)
		if _, werr := w.Write(internalServerErrorJson); werr != nil {
			// log.Err(werr).Msg("error while writing response into http.ResponseWriter")
		}
		return
	}

	w.WriteHeader(p.chooseStatusCode(resp.StatusCode()))
	w.Header().Add("X-From-Http-Cache-Proxy", "true")
	w.Header().Add("X-Hit-Http-Cache-Proxy", utils.BoolToString(isHit))
	if _, werr := w.Write(resp.GetBody()); werr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		// log.Err(werr).Msg("error while writing response into http.ResponseWriter")
		return
	}
	for headerName, v := range resp.GetHeaders() {
		for _, headerValue := range v {
			w.Header().Add(headerName, headerValue)
		}
	}
	return
}

func (p *Plugin) chooseStatusCode(statusCode int) int {
	if statusCode > 0 {
		return statusCode
	} else {
		return http.StatusInternalServerError
	}
}

func (p *Plugin) extractRequest(r *http.Request) (*model.Request, error) {
	project := r.URL.Query().Get("project")
	if project == "" {
		return nil, errors.New("project is missing")
	}
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		return nil, errors.New("domain is missing")
	}
	language := r.URL.Query().Get("language")
	if language == "" {
		return nil, errors.New("language is missing")
	}
	choice := r.URL.Query().Get("choice")
	if choice == "" {
		return nil, errors.New("choice is missing")
	}
	return model.NewRequest(project, domain, language, choice), nil
}

var _ http.ResponseWriter = &captureResponseWriter{}

type captureResponseWriter struct {
	wrapped     http.ResponseWriter
	body        *bytes.Buffer
	statusCode  int
	headers     http.Header
	wroteHeader bool
}

func newCaptureResponseWriter(w http.ResponseWriter) *captureResponseWriter {
	return &captureResponseWriter{
		wrapped:    w,
		body:       new(bytes.Buffer),
		statusCode: http.StatusOK,
		headers:    make(http.Header),
	}
}

func (w *captureResponseWriter) Header() http.Header {
	// intercept and work with our copy of headers
	return w.headers
}

func (w *captureResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return // prevent double WriteHeader
	}
	w.statusCode = code
	w.wroteHeader = true

	// copy headers to the underlying writer before writing header
	for k, v := range w.headers {
		for _, vv := range v {
			w.wrapped.Header().Add(k, vv)
		}
	}
	w.wrapped.WriteHeader(code)
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	// ensure WriteHeader is called if not already done
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.body.Write(b) // Save to buffer
	return w.wrapped.Write(b)
}
