package model

import (
	"container/list"
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
)

const aggressiveBeta = 0.9 // aggressive revalidation (probably may be upped to 0.95 or 0.98-0.99)

var nameRegexp = regexp.MustCompile(`"name"\s*:\s*"([^"]*)"`)

type ResponseCreator func(ctx context.Context, req *Request) (statusCode int, body []byte, headers http.Header, err error)

type Response struct {
	mu                 *sync.RWMutex
	cfg                config.Config
	request            *Request // request for current response
	tags               []string // choice names as tags
	headers            http.Header
	statusCode         int
	body               []byte // raw body of response
	listElement        *list.Element
	creator            ResponseCreator
	revalidateInterval time.Duration
	revalidatedAt      time.Time // last revalidated timestamp
	revalidateBeta     float64
}

func (r *Response) StatusCode() int {
	r.mu.RUnlock()
	defer r.mu.RUnlock()
	return r.statusCode
}

func NewResponse(
	headers http.Header,
	statusCode int,
	req *Request,
	body []byte,
	creator ResponseCreator,
	revalidateInterval time.Duration,
	revalidateBeta float64,
) (*Response, error) {
	tags, err := ExtractTags(req.GetChoice())
	if err != nil {
		return nil, fmt.Errorf("cannot extract tags from choice: %s", err.Error())
	}
	return &Response{
		mu:                 &sync.RWMutex{},
		request:            req,
		tags:               tags,
		statusCode:         statusCode,
		headers:            headers,
		body:               body,
		creator:            creator,
		revalidateInterval: revalidateInterval,
		revalidateBeta:     revalidateBeta,
		revalidatedAt:      time.Now(),
	}, nil
}
func (r *Response) Revalidate(ctx context.Context) {
	var err error
	defer func() {
		if err != nil {
			// log.Debug().Msg("revalidation failed")
		} else {
			//// log.Debug().Msg("success revalidated")
			// log.Debug().Msg("success revalidated")
		}
	}()

	r.mu.RLock()
	req := r.request
	creator := r.creator
	r.mu.RUnlock()

	statusCode, bytes, headers, err := creator(ctx, req)
	if err != nil {
		return
	}

	r.mu.Lock()
	r.revalidatedAt = time.Now()
	r.body = bytes
	r.headers = headers
	r.statusCode = statusCode
	r.mu.Unlock()
	return
}
func (r *Response) ShouldBeRevalidated() bool {
	r.mu.RLock()
	revalidatedAt := r.revalidatedAt
	revalidatedInterval := r.revalidateInterval
	// beta = 0.5 — обычно хорошее стартовое значение
	// beta = 1.0 — агрессивное обновление
	// beta = 0.0 — отключает бета-обновление полностью
	beta := r.revalidateBeta
	statusCode := r.statusCode
	r.mu.RUnlock()

	if statusCode != http.StatusOK {
		beta = aggressiveBeta
	} else {
		beta = r.revalidateBeta
	}

	return r.shouldRevalidateBeta(revalidatedAt, revalidatedInterval, beta)
}
func (r *Response) shouldRevalidateBeta(revalidatedAt time.Time, revalidateInterval time.Duration, beta float64) bool {
	age := time.Since(revalidatedAt)

	if age < time.Duration(beta*float64(revalidateInterval)) {
		return false
	}

	if age >= revalidateInterval {
		return true
	}

	return rand.Float64() >= math.Exp(-beta*float64(age)/float64(revalidateInterval))
}

func (r *Response) GetRequest() *Request {
	r.mu.RLock()
	req := r.request
	r.mu.RUnlock()
	return req
}
func (r *Response) GetListElement() *list.Element {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.listElement
}
func (r *Response) SetListElement(el *list.Element) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listElement = el
}
func (r *Response) GetBody() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.body
}
func (r *Response) GetHeaders() http.Header {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.headers
}
func (r *Response) Size() uintptr {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var size int

	// === http.Response data ===
	// headers
	for key, values := range r.headers {
		size += len(key)
		for _, val := range values {
			size += len(val)
		}
	}
	// body
	size += len(r.body)
	// statusCode: 4 bytes (int)
	size += 4

	// === Request ===
	if r.request != nil {
		size += len(r.request.project)
		size += len(r.request.domain)
		size += len(r.request.language)
		size += len(r.request.choice)
		size += len(r.request.uniqueString)
		size += len(r.request.uniqueQuery)
		// uniqueKey: 8 bytes (uint64)
		size += 8
	}

	// === Response ===

	// tags slice (array of strings)
	for _, tag := range r.tags {
		size += len(tag)
	}

	// Timestamps: 2 x time.Time (assume 15 bytes each = 30 bytes total)
	size += 15 * 2

	// Floats (revalidateBeta): 8 bytes
	size += 8

	// Durations: 8 bytes
	size += 8

	return uintptr(size)
}

func ExtractTags(choice string) ([]string, error) {
	matches := nameRegexp.FindAllStringSubmatch(choice, -1)
	tags := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			tags = append(tags, match[1])
		}
	}

	return tags, nil
}
