package model

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	wheelmodel "github.com/Borislavv/traefik-http-cache-plugin/pkg/wheel/model"
	"math"
	"math/rand/v2"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"
)

const gzipThreshold = 1024

var (
	dataPool = synced.NewBatchPool[*Data](synced.PreallocationBatchSize, func() *Data {
		return new(Data)
	})
	responsePool = synced.NewBatchPool[*Response](synced.PreallocationBatchSize, func() *Response {
		return &Response{
			data:          &atomic.Pointer[Data]{},
			request:       &atomic.Pointer[Request]{},
			lruListElem:   &atomic.Pointer[list.Element[*Response]]{},
			timeWheelElem: &atomic.Pointer[list.Element[wheelmodel.Spoke]]{},
		}
	})
	gzipBufferPool = synced.NewBatchPool[*bytes.Buffer](synced.PreallocationBatchSize, func() *bytes.Buffer {
		return new(bytes.Buffer)
	})
	gzipWriterPool = synced.NewBatchPool[*gzip.Writer](synced.PreallocationBatchSize, func() *gzip.Writer {
		w, err := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		if err != nil {
			panic("failed to init. gzip writer: " + err.Error())
		}
		return w
	})
)

type Data struct {
	statusCode int
	headers    http.Header

	body      []byte
	releaseFn synced.FreeResourceFunc
}

func NewData(statusCode int, headers http.Header, body []byte, releaseBody synced.FreeResourceFunc) *Data {
	data := dataPool.Get()
	*data = Data{
		headers:    headers,
		statusCode: statusCode,
		releaseFn:  releaseBody,
	}

	if len(body) > gzipThreshold {
		gzipper := gzipWriterPool.Get()
		defer gzipWriterPool.Put(gzipper)

		buf := gzipBufferPool.Get()
		gzipper.Reset(buf)
		buf.Reset()

		_, err := gzipper.Write(body)
		if err == nil && gzipper.Close() == nil {
			headers.Set("Content-Encoding", "gzip")
			data.body = buf.Bytes()
		} else {
			data.body = body
		}

		data.releaseFn = func() {
			releaseBody()
			buf.Reset()
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
func (d *Data) clear() {
	d.statusCode = 0
	d.headers = nil
	d.body = nil
	d.releaseFn = nil
}
func (d *Data) Release() {
	d.releaseFn()
	d.clear()
	dataPool.Put(d)
}

// Response weight is 80 bytes
type Response struct {
	request *atomic.Pointer[Request]

	data          *atomic.Pointer[Data]
	lruListElem   *atomic.Pointer[list.Element[*Response]]
	timeWheelElem *atomic.Pointer[list.Element[wheelmodel.Spoke]]

	revalidator func(ctx context.Context) (data *Data, err error)

	refCount int64
	isDoomed int64

	beta               int64 // float64(0.7) * 100 = 70
	maxStaleDuration   int64 // UnixNano
	revalidateInterval int64 // UnixNano
	revalidatedAt      int64 // UnixNano
}

func NewResponse(data *Data, req *Request, cfg *config.Config, revalidator func(ctx context.Context) (data *Data, err error)) (*Response, error) {
	return responsePool.Get().init().clear().setUp(cfg, data, req, revalidator), nil
}

func (r *Response) init() *Response {
	if r.request == nil {
		r.request = &atomic.Pointer[Request]{}
	}
	if r.data == nil {
		r.data = &atomic.Pointer[Data]{}
	}
	if r.lruListElem == nil {
		r.lruListElem = &atomic.Pointer[list.Element[*Response]]{}
	}
	return r
}

func (r *Response) setUp(cfg *config.Config, data *Data, req *Request, revalidator func(ctx context.Context) (data *Data, err error)) *Response {
	r.data.Store(data)
	r.request.Store(req)
	r.revalidator = revalidator
	r.revalidatedAt = time.Now().UnixNano()
	r.beta = int64(cfg.RevalidateBeta * 100)
	r.revalidateInterval = cfg.RevalidateInterval.Nanoseconds()
	r.maxStaleDuration = cfg.RefreshDurationThreshold.Nanoseconds()
	return r
}

func (r *Response) clear() *Response {
	r.revalidator = nil
	r.refCount = 0
	r.isDoomed = 0
	r.revalidatedAt = 0
	r.maxStaleDuration = 0
	r.revalidateInterval = 0
	r.beta = 0
	r.timeWheelElem.Store(nil)
	r.lruListElem.Store(nil)
	r.request.Store(nil)
	r.data.Store(nil)
	return r
}
func (r *Response) ToQuery() []byte {
	return r.request.Load().ToQuery()
}
func (r *Response) Key() uint64 {
	return atomic.LoadUint64(&r.request.Load().key)
}
func (r *Response) ShardKey() uint64 {
	return atomic.LoadUint64(&r.request.Load().shardKey)
}
func (r *Response) IsDoomed() bool {
	return atomic.LoadInt64(&r.isDoomed) == 1
}
func (r *Response) MarkAsDoomed() bool {
	return atomic.CompareAndSwapInt64(&r.isDoomed, 0, 1)
}
func (r *Response) RefCount() int64 {
	return atomic.LoadInt64(&r.refCount)
}
func (r *Response) IncRefCount() int64 {
	return atomic.AddInt64(&r.refCount, 1)
}
func (r *Response) DecRefCount() int64 {
	return atomic.AddInt64(&r.refCount, -1)
}
func (r *Response) CASRefCount(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&r.refCount, old, new)
}
func (r *Response) StoreRefCount(new int64) {
	atomic.StoreInt64(&r.refCount, new)
}

func (r *Response) ShouldRefresh() bool {
	age := time.Since(time.Unix(0, atomic.LoadInt64(&r.revalidatedAt))).Nanoseconds()
	if age > r.maxStaleDuration {
		return rand.Float64() >= math.Exp((float64(-r.beta)/100)*float64(age)/float64(r.revalidateInterval))
	}
	return false
}

func (r *Response) Revalidate(ctx context.Context) error {
	data, err := r.revalidator(ctx)
	if err != nil {
		return err
	}
	r.data.Store(data)
	atomic.StoreInt64(&r.revalidatedAt, time.Now().UnixNano())
	return nil
}

func (r *Response) Request() *Request {
	return r.request.Load()
}

func (r *Response) ShardListElement() any {
	return r.lruListElem.Load()
}

func (r *Response) LruListElement() *list.Element[*Response] {
	return r.lruListElem.Load()
}

func (r *Response) SetLruListElement(el *list.Element[*Response]) {
	r.lruListElem.Store(el)
}

func (r *Response) WheelListElement() *list.Element[wheelmodel.Spoke] {
	return r.timeWheelElem.Load()
}

func (r *Response) StoreWheelListElement(elem *list.Element[wheelmodel.Spoke]) {
	r.timeWheelElem.Store(elem)
}

func (r *Response) Data() *Data {
	return r.data.Load()
}

func (r *Response) Body() []byte {
	return r.data.Load().Body()
}

func (r *Response) Headers() http.Header {
	return r.data.Load().Headers()
}

func (r *Response) RevalidatedAt() time.Time {
	return time.Unix(0, atomic.LoadInt64(&r.revalidatedAt))
}
func (r *Response) NativeRevalidatedAt() int64 {
	return atomic.LoadInt64(&r.revalidatedAt)
}
func (r *Response) NativeRevalidateInterval() int64 {
	return atomic.LoadInt64(&r.revalidateInterval)
}
func (r *Response) Size() uintptr {
	var size = int(unsafe.Sizeof(r))

	// calc dynamic resp fields weight
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

	// calc dynamic req fields weight
	req := r.Request()
	if req != nil {
		size += int(unsafe.Sizeof(req))
		size += len(req.project)
		size += len(req.domain)
		size += len(req.language)
		for _, tag := range req.tags {
			size += len(tag)
		}
	}

	return uintptr(size)
}

func (r *Response) Release() bool {
	r.data.Load().Release()
	r.request.Load().Release()
	r.clear()
	responsePool.Put(r)
	return true
}
