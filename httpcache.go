package httpcache

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
)

type Config struct {
	EnableCache bool `json:"enableCache,omitempty"`
}

func CreateConfig() *Config {
	return &Config{}
}

type LogPlugin struct {
	next        http.Handler
	name        string
	enableCache bool

	mu    sync.RWMutex
	cache map[string]*cachedResponse
}

type cachedResponse struct {
	statusCode int
	headers    http.Header
	body       []byte
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &LogPlugin{
		next:        next,
		name:        name,
		enableCache: config.EnableCache,
		cache:       make(map[string]*cachedResponse),
	}, nil
}

func (p *LogPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	key := req.Method + ":" + req.URL.String()

	if p.enableCache {
		p.mu.RLock()
		cached, ok := p.cache[key]
		p.mu.RUnlock()

		if ok {
			fmt.Printf("[Plugin %s] Serving from cache: %s, content: %+v\n", p.name, key, cached)
			//fmt.Printf("[Plugin %s] Serving from cache: %s", p.name, key)
			for k, vv := range cached.headers {
				for _, v := range vv {
					rw.Header().Add(k, v)
				}
			}
			rw.WriteHeader(cached.statusCode)
			if cached.statusCode != 200 {
				fmt.Sprintf("NON SUCCESS RESPONSE: %+v\n", cached)
				panic("!!!")
			}
			rw.Write(cached.body)
			return
		}
	}

	respWriter := newCaptureResponseWriter(rw)

	p.next.ServeHTTP(respWriter, req)

	if p.enableCache && respWriter.statusCode == http.StatusOK {
		p.mu.Lock()
		p.cache[key] = &cachedResponse{
			statusCode: respWriter.statusCode,
			headers:    cloneHeaders(respWriter.Header()),
			body:       respWriter.body.Bytes(),
		}
		p.mu.Unlock()
		//fmt.Printf("[Plugin %s] Cached response: %s, cachedResponse: %+v\n", p.name, key, p.cache[key])
		//fmt.Printf("[Plugin %s] Cached response: %s\n", p.name, key)
	}
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
	// Intercept and work with our copy of headers
	return w.headers
}

func (w *captureResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return // Prevent double WriteHeader
	}
	w.statusCode = code
	w.wroteHeader = true

	// Copy headers to the underlying writer before writing header
	for k, vv := range w.headers {
		for _, v := range vv {
			w.wrapped.Header().Add(k, v)
		}
	}
	w.wrapped.WriteHeader(code)
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	// Ensure WriteHeader is called if not already done
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.body.Write(b) // Save to buffer
	return w.wrapped.Write(b)
}

func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header)
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
	return dst
}
