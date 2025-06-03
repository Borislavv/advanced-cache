package model

import (
	"container/list"
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/rs/zerolog/log"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/buger/jsonparser"
)

const (
	nameToken      = "name"
	aggressiveBeta = 0.9 // aggressive revalidation (probably may be upped to 0.95 or 0.98-0.99)
)

type ResponseCreator = func(ctx context.Context) (data *Data, err error)

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
	tags []string
	/* immutable */
	cfg *config.Config
	/* immutable */
	creator ResponseCreator
}

func (d *Datum) Headers() http.Header {
	return d.headers
}

func (d *Datum) StatusCode() int {
	return d.statusCode
}

func (d *Datum) Body() []byte {
	return d.body
}

func NewResponse(
	data *Data,
	req *Request,
	cfg *config.Config,
	creator ResponseCreator,
) (*Response, error) {
	tags, err := ExtractTags(req.GetChoice())
	if err != nil {
		return nil, fmt.Errorf("cannot extract tags from choice: %w", err)
	}
	resp := &Response{
		cfg:     cfg,
		tags:    tags,
		creator: creator,
	}
	resp.data.Store(data)
	resp.request.Store(req)
	resp.revalidatedAt.Store(time.Now().UnixNano())
	return resp, nil
}

func (r *Response) Revalidate(ctx context.Context) {
	data, err := r.creator(ctx)
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

func (r *Response) Size() uintptr {
	return unsafe.Sizeof(*r)
}

func ExtractTags(choice string) ([]string, error) {
	var names []string
	err := jsonparser.ObjectEach([]byte(choice), func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if string(key) == nameToken {
			names = append(names, string(value))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return names, nil
}
