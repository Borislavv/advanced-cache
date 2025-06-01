package middleware

import (
	"bytes"
	"context"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"net/http"
	"time"
)

var internalServerErrorJson = []byte(`{"error": {"message": "Internal server error."}}`)

type Config struct {
	config.Storage
	config.Response
}

func CreateConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Err(err).Msg(".env file not found, skipping")
	}
	viper.AutomaticEnv()
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Err(err).Msg("failed to unmarshal config")
	}
	return cfg
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
		seoRepo: repository.NewSeo(),
		storage: storage.New(config.Storage),
	}
}

func (p *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := p.extractRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if _, werr := w.Write(internalServerErrorJson); werr != nil {
			log.Err(err).Msg("error while writing response into http.ResponseWriter")
		}
		return
	}

	if resp, found := p.storage.Get(req); found {
		w.WriteHeader(http.StatusOK)
		if _, werr := w.Write(resp.GetBody()); werr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Err(werr).Msg("error while writing response into http.ResponseWriter")
			return
		}
		for headerName, v := range resp.GetHeaders() {
			for _, headerValue := range v {
				w.Header().Add(headerName, headerValue)
			}
		}
		return
	}

	clonedWriter := newCaptureResponseWriter(w)

	p.next.ServeHTTP(clonedWriter, r)

	if clonedWriter.statusCode != http.StatusOK {
		return
	}

	ctx, cancel := context.WithTimeout(p.ctx, time.Millisecond*400)
	defer cancel()

	resp, err := model.NewResponse(p.config.Response, clonedWriter.Header().Clone(), req, clonedWriter.body.Bytes(), p.seoRepo)
	if err != nil {
		log.Err(err).Msg("failed to make response")
		w.WriteHeader(http.StatusInternalServerError)
		if _, werr := w.Write(internalServerErrorJson); werr != nil {
			log.Err(werr).Msg("error while writing response into http.ResponseWriter")
		}
		return
	}

	p.storage.Set(ctx, resp)
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
