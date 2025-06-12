package model

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	timemodel "github.com/Borislavv/traefik-http-cache-plugin/pkg/time/model"
	"math"
	"math/rand/v2"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"
)

const gzipThreshold = 1024 // Minimum body size to apply gzip compression

// -- Internal pools for efficient memory management --

var (
	dataPool = synced.NewBatchPool[*Data](synced.PreallocationBatchSize, func() *Data {
		return new(Data)
	})
	responsePool = synced.NewBatchPool[*Response](synced.PreallocationBatchSize, func() *Response {
		return &Response{
			data:          &atomic.Pointer[Data]{},
			request:       &atomic.Pointer[Request]{},
			lruListElem:   &atomic.Pointer[list.Element[*Response]]{},
			timeWheelElem: &atomic.Pointer[list.Element[timemodel.Spoke]]{},
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

// Data is the actual payload (status, headers, body) stored in the cache.
type Data struct {
	statusCode int
	headers    http.Header
	body       []byte
	releaseFn  synced.FreeResourceFunc // To release memory/buffer if pooled
}

// NewData creates a new Data object, compressing body with gzip if large enough.
// Uses memory pools for buffer and writer to minimize allocations.
func NewData(statusCode int, headers http.Header, body []byte, releaseBody synced.FreeResourceFunc) *Data {
	data := dataPool.Get()
	*data = Data{
		headers:    headers,
		statusCode: statusCode,
		releaseFn:  releaseBody,
	}

	// Compress body if it's large enough for gzip to help
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

// Headers returns the response headers.
func (d *Data) Headers() http.Header { return d.headers }

// StatusCode returns the HTTP status code.
func (d *Data) StatusCode() int { return d.statusCode }

// Body returns the response body (possibly gzip-compressed).
func (d *Data) Body() []byte { return d.body }

// clear zeroes the Data fields before pooling.
func (d *Data) clear() {
	d.statusCode = 0
	d.headers = nil
	d.body = nil
	d.releaseFn = nil
}

// Release calls releaseFn and returns the Data to the pool.
func (d *Data) Release() {
	d.releaseFn()
	d.clear()
	dataPool.Put(d)
}

// Response is the main cache object, holding the request, payload, metadata, and list pointers.
type Response struct {
	request       *atomic.Pointer[Request]                       // Associated request
	data          *atomic.Pointer[Data]                          // Cached data
	lruListElem   *atomic.Pointer[list.Element[*Response]]       // Pointer for LRU list (per-shard)
	timeWheelElem *atomic.Pointer[list.Element[timemodel.Spoke]] // Pointer for wheel/bucket-based background refresh

	revalidator func(ctx context.Context) (data *Data, err error) // Closure for refresh/revalidation

	refCount int64 // refCount for concurrent/lifecycle management
	isDoomed int64 // "Doomed" flag for objects marked for delete but still referenced

	beta               int64 // e.g. 0.7*100 = 70, parameter for probabilistic refresh
	maxStaleDuration   int64 // How long to keep without refresh (nanoseconds)
	revalidateInterval int64 // Interval for background revalidation (nanoseconds)
	revalidatedAt      int64 // Last revalidated time (nanoseconds since epoch)
}

// NewResponse constructs a new Response using memory pools and sets up all fields.
func NewResponse(
	data *Data,
	req *Request,
	cfg *config.Config,
	revalidator func(ctx context.Context) (data *Data, err error),
) (*Response, error) {
	return responsePool.Get().init().clear().setUp(cfg, data, req, revalidator), nil
}

// init ensures all pointers are non-nil after pool Get.
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

// setUp stores the Data, Request, and config-driven fields into the Response.
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

// clear resets the Response for re-use from the pool.
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

// --- Response API ---

// ToQuery returns the query representation of the request.
func (r *Response) ToQuery() []byte {
	return r.request.Load().ToQuery()
}

// Key returns the key of the associated request.
func (r *Response) Key() uint64 {
	return atomic.LoadUint64(&r.request.Load().key)
}

// ShardKey returns the shard key of the associated request.
func (r *Response) ShardKey() uint64 {
	return atomic.LoadUint64(&r.request.Load().shardKey)
}

// IsDoomed returns true if this object is scheduled for deletion.
func (r *Response) IsDoomed() bool {
	return atomic.LoadInt64(&r.isDoomed) == 1
}

// MarkAsDoomed marks the response as scheduled for delete, only if not already doomed.
func (r *Response) MarkAsDoomed() bool {
	return atomic.CompareAndSwapInt64(&r.isDoomed, 0, 1)
}

// RefCount returns the current refcount.
func (r *Response) RefCount() int64 {
	return atomic.LoadInt64(&r.refCount)
}

// IncRefCount increments the refcount.
func (r *Response) IncRefCount() int64 {
	return atomic.AddInt64(&r.refCount, 1)
}

// DecRefCount decrements the refcount.
func (r *Response) DecRefCount() int64 {
	return atomic.AddInt64(&r.refCount, -1)
}

// CASRefCount performs a CAS on the refcount.
func (r *Response) CASRefCount(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&r.refCount, old, new)
}

// StoreRefCount stores a new refcount value directly.
func (r *Response) StoreRefCount(new int64) {
	atomic.StoreInt64(&r.refCount, new)
}

// ShouldRefresh implements probabilistic refresh logic ("beta" algorithm).
// Returns true if the entry is stale and, with a probability proportional to its staleness, should be refreshed now.
func (r *Response) ShouldRefresh() bool {
	if atomic.LoadInt64(&r.isDoomed) == 1 {
		return false
	}
	age := time.Since(time.Unix(0, atomic.LoadInt64(&r.revalidatedAt))).Nanoseconds()
	if age > r.maxStaleDuration {
		return rand.Float64() >= math.Exp((float64(-r.beta)/100)*float64(age)/float64(r.revalidateInterval))
	}
	return false
}

// Revalidate calls the revalidator closure to fetch fresh data and updates the timestamp.
func (r *Response) Revalidate(ctx context.Context) error {
	data, err := r.revalidator(ctx)
	if err != nil {
		return err
	}
	r.data.Store(data)
	atomic.StoreInt64(&r.revalidatedAt, time.Now().UnixNano())
	return nil
}

// Request returns the request pointer.
func (r *Response) Request() *Request {
	return r.request.Load()
}

// ShardListElement returns the LRU list element (for cache eviction).
func (r *Response) ShardListElement() any {
	return r.lruListElem.Load()
}

// LruListElement returns the LRU list element pointer (for LRU cache management).
func (r *Response) LruListElement() *list.Element[*Response] {
	return r.lruListElem.Load()
}

// SetLruListElement sets the LRU list element pointer.
func (r *Response) SetLruListElement(el *list.Element[*Response]) {
	r.lruListElem.Store(el)
}

// WheelListElement returns the wheel list element pointer (for background refresh scheduling).
func (r *Response) WheelListElement() *list.Element[timemodel.Spoke] {
	return r.timeWheelElem.Load()
}

// StoreWheelListElement sets the wheel list element pointer.
func (r *Response) StoreWheelListElement(elem *list.Element[timemodel.Spoke]) {
	r.timeWheelElem.Store(elem)
}

// Data returns the underlying Data payload.
func (r *Response) Data() *Data {
	return r.data.Load()
}

// Body returns the response body.
func (r *Response) Body() []byte {
	return r.data.Load().Body()
}

// Headers returns the HTTP headers.
func (r *Response) Headers() http.Header {
	return r.data.Load().Headers()
}

// RevalidatedAt returns the last revalidation time (as time.Time).
func (r *Response) RevalidatedAt() time.Time {
	return time.Unix(0, atomic.LoadInt64(&r.revalidatedAt))
}

// NativeRevalidatedAt returns the last revalidation time (as int64).
func (r *Response) NativeRevalidatedAt() int64 {
	return atomic.LoadInt64(&r.revalidatedAt)
}

// NativeRevalidateInterval returns the revalidation interval (as int64).
func (r *Response) NativeRevalidateInterval() int64 {
	return atomic.LoadInt64(&r.revalidateInterval)
}

// Size estimates the in-memory size of this response (including dynamic fields).
func (r *Response) Size() uintptr {
	var size = int(unsafe.Sizeof(r))

	// Account for dynamic response fields
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

	// Account for dynamic request fields
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

// Release releases the associated Data and Request, resets the Response, and returns it to the pool.
func (r *Response) Release() bool {
	r.data.Load().Release()
	r.request.Load().Release()
	r.clear()
	responsePool.Put(r)
	return true
}
