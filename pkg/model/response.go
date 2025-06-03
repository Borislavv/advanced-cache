package model

import (
	"container/list"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/rs/zerolog/log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var ResponsePool = &sync.Pool{
	New: func() interface{} {
		return &Response{}
	},
}

type ResponseRevalidator = func(ctx context.Context) (data *Data, err error)

type Data struct {
	headers    http.Header
	statusCode int
	body       []byte
}

func NewData(statusCode int, headers http.Header, body []byte) *Data {
	return &Data{
		headers:    headers,
		statusCode: statusCode,
		body:       body,
	}
}

func (d *Data) Headers() http.Header { return d.headers }
func (d *Data) StatusCode() int      { return d.statusCode }
func (d *Data) Body() []byte         { return d.body }

// Response weight is 80 bytes
type Response struct {
	/* mutable (pointer change but data are immutable) */
	data atomic.Pointer[Data]
	/* mutable (pointer change but data are immutable) */
	request atomic.Pointer[Request]
	/* mutable */
	revalidatedAt atomic.Int64 // UnixNano
	/* mutable */
	listElement atomic.Pointer[list.Element]

	/* immutable */
	tags [][]byte
	/* immutable */
	cfg *config.Config
	/* immutable */
	revalidator ResponseRevalidator
}

func NewResponse(
	data *Data,
	req *Request,
	cfg *config.Config,
	revalidator ResponseRevalidator,
) (*Response, error) {
	resp, ok := ResponsePool.Get().(*Response)
	if !ok {
		panic("pool must contains only *model.Response")
	}
	if resp.cfg == nil {
		resp.cfg = cfg
	}
	resp.request.Store(req)
	resp.revalidator = revalidator
	resp.tags = req.GetTags()
	resp.data.Store(data)
	resp.revalidatedAt.Store(time.Now().UnixNano())
	resp.listElement.Store(nil)
	return resp, nil
}

func (r *Response) Revalidate(ctx context.Context) {
	data, err := r.revalidator(ctx)
	if err != nil {
		log.Err(err).Msg("error occurred while revalidating item")
		return
	}
	r.data.Store(data)
	r.revalidatedAt.Store(time.Now().UnixNano())
}

func (r *Response) ShouldBeRevalidated() bool {
	age := time.Since(time.Unix(0, r.revalidatedAt.Load()))
	if age < time.Duration(float64(r.cfg.RevalidateInterval)*r.cfg.RevalidateBeta) {
		return false
	}
	if age >= r.cfg.RevalidateInterval {
		return true
	}
	return rand.Float64() >= math.Exp(-r.cfg.RevalidateBeta*float64(age)/float64(r.cfg.RevalidateInterval))
}

func (r *Response) GetRequest() *Request {
	return r.request.Load()
}

func (r *Response) GetListElement() *list.Element {
	return r.listElement.Load()
}

func (r *Response) SetListElement(el *list.Element) {
	r.listElement.Store(el)
}

func (r *Response) GetData() *Data {
	return r.data.Load()
}

func (r *Response) SetData(d *Data) {
	r.data.Store(d)
}

func (r *Response) GetBody() []byte {
	return r.data.Load().Body()
}

func (r *Response) GetHeaders() http.Header {
	return r.data.Load().Headers()
}

func (r *Response) GetRevalidatedAt() time.Time {
	return time.Unix(0, r.revalidatedAt.Load())
}
func (r *Response) Size() uintptr {
	var size = 80

	data := r.data.Load()
	if data != nil {
		for key, values := range data.headers {
			size += len(key)
			for _, val := range values {
				size += len(val)
			}
		}
		size += len(data.body)
		// statusCode: 4 bytes (int)
		size += 4
	}

	req := r.GetRequest()
	if req != nil {
		size += len(req.project)
		size += len(req.domain)
		size += len(req.language)
		for _, tag := range req.tags {
			size += len(tag)
		}
		size += 24 // tags field is slice
		size += len(req.uniqueQuery)
		size += 8 // req.uniqueKey: 8 bytes (uint64)
	}
	size += 47 // data, revalidatedAt, revalidateBeta, cfg, listElem,

	return uintptr(size)
}
