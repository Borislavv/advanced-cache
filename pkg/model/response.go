package model

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"math"
	"math/rand/v2"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"
)

const gzipThreshold = 1024 // 1KB

var (
	DataPool = synced.NewBatchPool[*Data](synced.PreallocationBatchSize, func() *Data {
		return new(Data)
	})
	ResponsePool = synced.NewBatchPool[*Response](synced.PreallocationBatchSize, func() *Response {
		return new(Response)
	})
	gzipBufferPool = synced.NewBatchPool[*bytes.Buffer](synced.PreallocationBatchSize, func() *bytes.Buffer {
		return new(bytes.Buffer)
	})
	gzipWriterPool = synced.NewBatchPool[*gzip.Writer](synced.PreallocationBatchSize, func() *gzip.Writer {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return w
	})
)

type ResponseRevalidator = func(ctx context.Context) (data *Data, err error)

type Data struct {
	statusCode int
	body       []byte
	headers    http.Header
	freeBody   synced.FreeResourceFunc
}

func NewData(statusCode int, headers http.Header, body []byte, freeBody synced.FreeResourceFunc) *Data {
	data := DataPool.Get()
	*data = Data{
		headers:    headers,
		statusCode: statusCode,
		freeBody:   freeBody,
	}

	if len(body) > gzipThreshold {
		gzipper := gzipWriterPool.Get()
		defer gzipWriterPool.Put(gzipper)

		buf := gzipBufferPool.Get()
		gzipper.Reset(buf)

		_, err := gzipper.Write(body)
		if err == nil {
			_ = gzipper.Close()
			data.body = buf.Bytes()
			headers.Set("Content-Encoding", "gzip")
		} else {
			data.body = body // fallback
		}

		data.freeBody = func() {
			freeBody()
			gzipBufferPool.Put(buf)
		}
	} else {
		data.body = body
	}

	return data
}

func (d *Data) Headers() http.Header { return d.headers }
func (d *Data) StatusCode() int      { return d.statusCode }
func (d *Data) Body() []byte         { return d.body }
func (d *Data) Free() {
	d.freeBody()
	DataPool.Put(d)
}

type Response struct {
	cfg           *config.Config // does not change
	data          *atomic.Pointer[Data]
	request       *atomic.Pointer[Request]
	revalidatedAt *atomic.Int64
	listElement   *atomic.Pointer[list.Element[*Request]]
	shardKey      *atomic.Uint64
	revalidator   *atomic.Pointer[ResponseRevalidator]
}

func NewResponse(
	data *Data,
	req *Request,
	cfg *config.Config,
	revalidator ResponseRevalidator,
) (*Response, error) {
	resp := ResponsePool.Get()

	if resp.data == nil {
		resp.data = &atomic.Pointer[Data]{}
	}
	if resp.request == nil {
		resp.request = &atomic.Pointer[Request]{}
	}
	if resp.revalidatedAt == nil {
		resp.revalidatedAt = &atomic.Int64{}
	}
	if resp.listElement == nil {
		resp.listElement = &atomic.Pointer[list.Element[*Request]]{}
	}
	if resp.shardKey == nil {
		resp.shardKey = &atomic.Uint64{}
	}
	if resp.revalidator == nil {
		resp.revalidator = &atomic.Pointer[ResponseRevalidator]{}
	}
	if resp.cfg == nil {
		resp.cfg = cfg
	}

	resp.data.Store(data)
	resp.request.Store(req)
	resp.revalidator.Store(&revalidator)
	resp.revalidatedAt.Store(time.Now().UnixNano())

	return resp, nil
}

func (r *Response) ShouldBeRevalidated(source *Response) bool {
	if source != nil {
		return false
	}
	age := time.Since(time.Unix(0, r.revalidatedAt.Load()))
	if age > r.cfg.RefreshEvictionDurationThreshold {
		return rand.Float64() >= math.Exp(-r.cfg.RevalidateBeta*float64(age)/float64(r.cfg.RevalidateInterval))
	}
	return false
}

func (r *Response) Revalidate(ctx context.Context) error {
	fn := r.revalidator.Load()
	if fn == nil || *fn == nil {
		return nil
	}
	data, err := (*fn)(ctx)
	if err != nil {
		return err
	}
	r.data.Store(data)
	r.revalidatedAt.Store(time.Now().UnixNano())
	return nil
}

func (r *Response) GetRequest() *Request {
	return r.request.Load()
}

func (r *Response) GetListElement() *list.Element[*Request] {
	return r.listElement.Load()
}

func (r *Response) SetListElement(el *list.Element[*Request]) {
	r.listElement.Store(el)
}

func (r *Response) GetShardKey() uint64 {
	return r.shardKey.Load()
}

func (r *Response) SetShardKey(shardKey uint64) {
	r.shardKey.Store(shardKey)
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
	var size = int(unsafe.Sizeof(r) + unsafe.Sizeof(r.data))

	data := r.data.Load()
	if data != nil {
		for key, values := range data.headers {
			size += len(key)
			for _, val := range values {
				size += len(val)
			}
		}
		size += len(data.body)
	}

	req := r.request.Load()
	if req != nil {
		size += int(unsafe.Sizeof(req))
		size += len(req.project)
		size += len(req.domain)
		size += len(req.language)
		for _, tag := range req.tags {
			size += len(tag)
		}
		size += int(unsafe.Sizeof(req.uniqueKey))
		if q := req.ToQuery(); q != nil {
			size += len(q)
		}
	}

	return uintptr(size)
}

func (r *Response) Free() {
	if d := r.data.Load(); d != nil {
		d.Free()
	}
	if req := r.request.Load(); req != nil {
		req.Free()
	}
	r.listElement.Store(nil)
	r.shardKey.Store(0)
	ResponsePool.Put(r)
}
