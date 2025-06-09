package model

import (
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

var ResponsePool = synced.NewBatchPool[*Response](synced.PreallocationBatchSize, func() *Response {
	return new(Response)
})

type ResponseRevalidator = func(ctx context.Context) (data *Data, err error)

type Data struct {
	statusCode int
	headers    http.Header

	body     []byte
	freeBody synced.FreeResourceFunc
}

func NewData(statusCode int, headers http.Header, body []byte, freeBody synced.FreeResourceFunc) *Data {
	return &Data{
		headers:    headers,
		statusCode: statusCode,
		body:       body,
		freeBody:   freeBody,
	}
}

func (d *Data) Headers() http.Header { return d.headers }
func (d *Data) StatusCode() int      { return d.statusCode }
func (d *Data) Body() []byte         { return d.body }

// Response weight is 80 bytes
type Response struct {
	/* mutable (pointer change but data are immutable) */
	data *atomic.Pointer[Data]
	/* mutable (pointer change but data are immutable) */
	request *atomic.Pointer[Request]
	/* mutable */
	revalidatedAt *atomic.Int64 // UnixNano
	/* mutable */
	listElement *atomic.Pointer[list.Element[*Request]]

	/* immutable */
	shardKey uint64
	/* immutable */
	cfg *config.Config
	/* immutable */
	revalidator ResponseRevalidator

	refCount    int64
	isReleasing int64
	isFreed     int64
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
	if resp.request == nil {
		resp.request = &atomic.Pointer[Request]{}
	}
	if resp.data == nil {
		resp.data = &atomic.Pointer[Data]{}
	}
	if resp.revalidatedAt == nil {
		resp.revalidatedAt = &atomic.Int64{}
		resp.revalidatedAt.Store(time.Now().UnixNano())
	}
	if resp.listElement == nil {
		resp.listElement = &atomic.Pointer[list.Element[*Request]]{}
	}

	resp.data.Store(data)
	resp.request.Store(req)
	resp.revalidator = revalidator

	return resp, nil
}

func (r *Response) StartReleasing() bool {
	return atomic.CompareAndSwapInt64(&r.isReleasing, 0, 1)
}
func (r *Response) StopReleasing() bool {
	return atomic.CompareAndSwapInt64(&r.isReleasing, 1, 0)
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
	data, err := r.revalidator(ctx)
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
	return time.Unix(0, r.revalidatedAt.Load())
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
	req := r.GetRequest()
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
	r.Free()
	return true
}

func (r *Response) Free() {
	if atomic.CompareAndSwapInt64(&r.isFreed, 0, 1) {
		r.data.Load().freeBody()
		r.request.Load().Free()

		r.revalidatedAt.Store(time.Now().UnixNano())
		atomic.StoreUint64(&r.shardKey, 0)
		r.listElement.Store(nil)
		r.request.Store(nil)
		r.data.Store(nil)

		r.StopReleasing()
		if !atomic.CompareAndSwapInt64(&r.isFreed, 1, 0) {
			panic("impossible (no one can be here else)")
		}
		ResponsePool.Put(r)
	}
}
