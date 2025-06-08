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
		return &Response{
			data:          &atomic.Pointer[Data]{},
			request:       &atomic.Pointer[Request]{},
			listElement:   &atomic.Pointer[list.Element[*Request]]{},
			revalidator:   &atomic.Pointer[ResponseRevalidator]{},
			refCounter:    1,
			revalidatedAt: 0,
			shardKey:      0,
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

type ResponseRevalidator = func(ctx context.Context) (data *Data, err error)

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
	data.body = data.body[:0]

	if len(body) > gzipThreshold {
		gzipper := gzipWriterPool.Get()
		defer gzipWriterPool.Put(gzipper)

		buf := gzipBufferPool.Get()
		gzipper.Reset(buf)

		_, err := gzipper.Write(body)
		if err == nil {
			_ = gzipper.Close()
			data.body = buf.Bytes()
			//data.body = append(data.body[:0], buf.Bytes()...)
			headers.Set("Content-Encoding", "gzip")
		} else {
			data.body = body
			//data.body = append(data.body, body...) // fallback
		}

		data.putBodyInPool = func() {
			freeBody()
			gzipBufferPool.Put(buf)
		}
	} else {
		//data.body = append(data.body, body...)
		data.body = body
	}

	return data
}

func (d *Data) Headers() http.Header { return d.headers }
func (d *Data) StatusCode() int      { return d.statusCode }
func (d *Data) Body() []byte         { return d.body }
func (d *Data) Free() {
	d.putBodyInPool()
	DataPool.Put(d)
}

type ResponseAcquireFn func()

type Response struct {
	cfg         *config.Config // does not change
	data        *atomic.Pointer[Data]
	request     *atomic.Pointer[Request]
	listElement *atomic.Pointer[list.Element[*Request]]
	revalidator *atomic.Pointer[ResponseRevalidator]

	refCounter    int64
	revalidatedAt int64
	isFreed       int32
	shardKey      uint64
}

func NewResponse(
	data *Data,
	req *Request,
	cfg *config.Config,
	revalidator ResponseRevalidator,
) (*Response, error) {
	resp := ResponsePool.Get()
	if resp.cfg == nil {
		resp.cfg = cfg
	}

	resp.data.Store(data)
	resp.request.Store(req)
	resp.revalidator.Store(&revalidator)
	resp.listElement.Store(nil)
	atomic.StoreInt64(&resp.revalidatedAt, time.Now().Unix())
	atomic.StoreInt32(&resp.isFreed, 0)
	atomic.StoreInt64(&resp.refCounter, 1)

	return resp, nil
}

func (r *Response) IsFreed() int32 {
	return atomic.LoadInt32(&r.isFreed)
}

func (r *Response) RefCount() int64 {
	return atomic.LoadInt64(&r.refCounter)
}

func (r *Response) IncRefCount() {
	atomic.AddInt64(&r.refCounter, 1)
}

func (r *Response) DcrRefCount() {
	atomic.AddInt64(&r.refCounter, -1)
}

func (r *Response) ShouldBeRevalidated(source *Response) bool {
	if source != nil {
		return false
	}
	age := time.Since(time.Unix(0, atomic.LoadInt64(&r.revalidatedAt)))
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
	return atomic.LoadUint64(&r.shardKey)
}

func (r *Response) SetShardKey(shardKey uint64) {
	atomic.StoreUint64(&r.shardKey, shardKey)
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
	return time.Unix(0, atomic.LoadInt64(&r.revalidatedAt))
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
	var project, domain, language, tags = r.GetRequest().query.Load().Values()
	if req != nil {
		size += int(unsafe.Sizeof(req))
		size += len(project)
		size += len(domain)
		size += len(language)
		for _, tag := range tags {
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
	r.data.Load().Free()
	r.request.Load().Free()
	ResponsePool.Put(r)
	atomic.StoreInt32(&r.isFreed, 1)
}
