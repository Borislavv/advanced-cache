package model

import (
	"bytes"
	"container/list"
	"context"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
)

const nameToken = "name"

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
	resp := &Response{
		cfg:         cfg,
		tags:        req.GetTags(),
		revalidator: revalidator,
	}
	resp.data.Store(data)
	resp.request.Store(req)
	resp.revalidatedAt.Store(time.Now().UnixNano())
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
	return unsafe.Sizeof(*r)
}

// ExtractTags - returns a slice with []byte("${choice name}").
func ExtractTags(args *fasthttp.Args) [][]byte {
	type entry struct {
		depth int
		value []byte
	}

	var entries []entry
	args.VisitAll(func(key, value []byte) {
		if !bytes.HasPrefix(key, choiceValue) || bytes.Equal(value, nullValue) {
			return
		}
		depth := bytes.Count(key, arrChoiceValue)
		entries = append(entries, entry{depth: depth, value: value})
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].depth < entries[j].depth
	})

	ordered := make([][]byte, 0, len(entries))
	for _, entryItem := range entries {
		ordered = append(ordered, entryItem.value)
	}

	return ordered
}
