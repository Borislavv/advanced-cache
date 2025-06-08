// response.go — high-performance, reference-counted immutable response snapshot.
// Thread-safe access via atomic.Pointer, lifecycle controlled via refCount + explicit Free.

package model

import (
	"bytes"
	"compress/gzip"
	"context"
	"math"
	"math/rand/v2"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
)

const gzipThreshold = 1024 // 1KB

var (
	DataPool = synced.NewBatchPool[*Data](synced.PreallocationBatchSize, func() *Data {
		return new(Data)
	})
	ResponsePool = synced.NewBatchPool[*Response](synced.PreallocationBatchSize, func() *Response {
		return &Response{
			data:        &atomic.Pointer[Data]{},
			request:     &atomic.Pointer[Request]{},
			listElement: &atomic.Pointer[list.Element[*Request]]{},
			revalidator: &atomic.Pointer[ResponseRevalidator]{},
			refCounter:  1,
		}
	})
	gzipBufferPool = synced.NewBatchPool[*bytes.Buffer](synced.PreallocationBatchSize, func() *bytes.Buffer {
		return new(bytes.Buffer)
	})
	gzipWriterPool = synced.NewBatchPool[*gzip.Writer](synced.PreallocationBatchSize, func() *gzip.Writer {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return w
	})
)

type ResponseRevalidator = func(ctx context.Context) (*Data, error)

// Data is immutable, safe to share between readers, released only when refCount == 0.
type Data struct {
	statusCode    int
	body          []byte
	headers       http.Header
	putBodyInPool synced.FreeResourceFunc
}

func NewData(statusCode int, headers http.Header, body []byte, freeBody synced.FreeResourceFunc) *Data {
	data := DataPool.Get()
	*data = Data{
		headers:       headers,
		statusCode:    statusCode,
		putBodyInPool: freeBody,
	}

	if len(body) > gzipThreshold {
		gzipper := gzipWriterPool.Get()
		defer gzipWriterPool.Put(gzipper)

		buf := gzipBufferPool.Get()
		buf.Reset()
		gzipper.Reset(buf)

		_, err := gzipper.Write(body)
		if err == nil {
			_ = gzipper.Close()
			data.body = append(data.body[:0], buf.Bytes()...)
			headers.Set("Content-Encoding", "gzip")
		} else {
			data.body = append(data.body[:0], body...)
		}

		data.putBodyInPool = func() {
			freeBody()
			buf.Reset()
			gzipBufferPool.Put(buf)
		}
	} else {
		data.body = append(data.body[:0], body...)
	}
	return data
}

func (d *Data) Headers() http.Header { return d.headers }
func (d *Data) StatusCode() int      { return d.statusCode }
func (d *Data) Body() []byte         { return d.body }

func (d *Data) Free() {
	d.putBodyInPool()
	d.body = d.body[:0]
	d.headers = nil
	d.putBodyInPool = nil
	DataPool.Put(d)
}

// Response is atomic, immutable snapshot used in LRU.
// It is released only when refCount == 0. Safe for concurrent reads.
type Response struct {
	cfg           *config.Config
	data          *atomic.Pointer[Data]
	request       *atomic.Pointer[Request]
	listElement   *atomic.Pointer[list.Element[*Request]]
	revalidator   *atomic.Pointer[ResponseRevalidator]
	refCounter    int64
	revalidatedAt int64
	isFreed       int32
	shardKey      uint64
}

func NewResponse(data *Data, req *Request, cfg *config.Config, revalidator ResponseRevalidator) (*Response, error) {
	resp := ResponsePool.Get()
	if resp.cfg == nil {
		resp.cfg = cfg
	}
	resp.data.Store(data)
	resp.request.Store(req)
	resp.revalidator.Store(&revalidator)
	resp.listElement.Store(nil)
	atomic.StoreInt64(&resp.revalidatedAt, time.Now().UnixNano())
	atomic.StoreInt64(&resp.refCounter, 1)
	atomic.StoreInt32(&resp.isFreed, 0)
	return resp, nil
}

func (r *Response) IncRefCount()                             { atomic.AddInt64(&r.refCounter, 1) }
func (r *Response) DcrRefCount()                             { atomic.AddInt64(&r.refCounter, -1) }
func (r *Response) RefCount() int64                          { return atomic.LoadInt64(&r.refCounter) }
func (r *Response) IsFreed() bool                            { return atomic.LoadInt32(&r.isFreed) == 1 }
func (r *Response) SetShardKey(id uint64)                    { atomic.StoreUint64(&r.shardKey, id) }
func (r *Response) GetShardKey() uint64                      { return atomic.LoadUint64(&r.shardKey) }
func (r *Response) GetData() *Data                           { return r.data.Load() }
func (r *Response) SetData(d *Data)                          { r.data.Store(d) }
func (r *Response) GetBody() []byte                          { return r.data.Load().Body() }
func (r *Response) GetHeaders() http.Header                  { return r.data.Load().Headers() }
func (r *Response) GetRequest() *Request                     { return r.request.Load() }
func (r *Response) GetListElement() *list.Element[*Request]  { return r.listElement.Load() }
func (r *Response) SetListElement(e *list.Element[*Request]) { r.listElement.Store(e) }

func (r *Response) GetRevalidatedAt() time.Time {
	return time.Unix(0, atomic.LoadInt64(&r.revalidatedAt))
}

func (r *Response) ShouldBeRevalidated(source *Response) bool {
	if source != nil {
		return false
	}
	age := time.Since(r.GetRevalidatedAt())
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
	atomic.StoreInt64(&r.revalidatedAt, time.Now().UnixNano())
	return nil
}

func (r *Response) Size() uintptr {
	size := unsafe.Sizeof(*r)
	d := r.GetData()
	if d != nil {
		for k, vals := range d.headers {
			size += uintptr(len(k))
			for _, v := range vals {
				size += uintptr(len(v))
			}
		}
		size += uintptr(len(d.body))
	}
	req := r.GetRequest()
	if req != nil {
		project, domain, language, tags := req.query.Load().Values()
		size += uintptr(len(project) + len(domain) + len(language))
		for _, tag := range tags {
			size += uintptr(len(tag))
		}
		if q := req.ToQuery(); q != nil {
			size += uintptr(len(q))
		}
	}
	return size
}

func (r *Response) Free() {
	r.data.Load().Free()
	r.request.Load().Free()
	ResponsePool.Put(r)
	atomic.StoreInt32(&r.isFreed, 1)
}
