package model

import (
	"container/list"
	"fmt"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/repository"
	"github.com/buger/jsonparser"
	"github.com/rs/zerolog/log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"unsafe"
)

const nameToken = "name"

type Response struct {
	*Datum
	mu            *sync.RWMutex
	cfg           config.Response
	seoRepo       repository.Seo
	request       *Request // request for current response
	tags          []string // choice names as tags
	listElement   *list.Element
	revalidatedAt time.Time // last revalidated timestamp
	revalidator   func() ([]byte, error)
	createdAt     time.Time
}

type Datum struct {
	headers http.Header
	body    []byte // raw body of response
}

func NewResponse(
	cfg config.Response,
	headers http.Header,
	req *Request,
	body []byte,
	revalidator func() ([]byte, error),
) (*Response, error) {
	tags, err := ExtractTags(req.GetChoice())
	if err != nil {
		return nil, fmt.Errorf("cannot extract tags from choice: %s", err.Error())
	}
	return &Response{
		mu:          &sync.RWMutex{},
		cfg:         cfg,
		request:     req,
		tags:        tags,
		revalidator: revalidator,
		Datum: &Datum{
			headers: headers,
			body:    body,
		},
		revalidatedAt: time.Now(),
		createdAt:     time.Now(),
	}, nil
}
func (r *Response) Revalidate() {
	defer log.Info().Msg("success revalidated")

	r.mu.RLock()
	revalidator := r.revalidator
	r.mu.RUnlock()

	data, err := revalidator()
	if err != nil {
		log.Warn().Err(err).Msg("revalidator failed")
		return
	}

	r.mu.Lock()
	r.revalidatedAt = time.Now()
	r.body = data
	r.mu.Unlock()
	return
}
func (r *Response) ShouldBeRevalidated() bool {
	r.mu.RLock()
	revalidatedAt := r.revalidatedAt
	revalidatedInterval := r.cfg.RevalidateInterval
	// beta = 0.5 — обычно хорошее стартовое значение
	// beta = 1.0 — агрессивное обновление
	// beta = 0.0 — отключает бета-обновление полностью
	beta := r.cfg.RevalidateBeta
	r.mu.RUnlock()

	return r.shouldRevalidateBeta(revalidatedAt, revalidatedInterval, beta)
}
func (r *Response) shouldRevalidateBeta(revalidatedAt time.Time, revalidateInterval time.Duration, beta float64) bool {
	now := time.Now()
	age := now.Sub(revalidatedAt)

	if age >= revalidateInterval {
		// properly expired, must be revalidated
		return true
	}

	// chance of prevent refresh
	probability := math.Exp(-beta * float64(age) / float64(revalidateInterval))
	rnd := rand.Float64() // [0.0, 1.0]

	return rnd >= probability
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

func (r *Response) GetDatum() *Datum {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Datum
}

func (r *Response) SetDatum(meta *Datum) {
	r.mu.Lock()
	r.Datum = meta
	r.mu.Unlock()
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
	return unsafe.Sizeof(*r)
}
func ExtractTags(choice string) ([]string, error) {
	names := make([]string, 0, 7)

	if err := jsonparser.ObjectEach([]byte(choice), func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if string(key) == nameToken {
			names = append(names, string(value))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return names, nil
}
