package model

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"io"
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
	dataPool = synced.NewBatchPool[*Data](synced.PreallocateBatchSize, func() *Data {
		return new(Data)
	})
	ResponsePool = synced.NewBatchPool[*Response](synced.PreallocateBatchSize, func() *Response {
		return &Response{
			data:        &atomic.Pointer[Data]{},
			request:     &atomic.Pointer[Request]{},
			lruListElem: &atomic.Pointer[list.Element[*Response]]{},
		}
	})
	gzipBufferPool = synced.NewBatchPool[*bytes.Buffer](synced.PreallocateBatchSize, func() *bytes.Buffer {
		return new(bytes.Buffer)
	})
	gzipWriterPool = synced.NewBatchPool[*gzip.Writer](synced.PreallocateBatchSize, func() *gzip.Writer {
		w, err := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		if err != nil {
			panic("failed to Init. gzip writer: " + err.Error())
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
	request     *atomic.Pointer[Request]                 // Associated request
	data        *atomic.Pointer[Data]                    // Cached data
	lruListElem *atomic.Pointer[list.Element[*Response]] // Pointer for LRU list (per-shard)

	revalidator func(ctx context.Context) (data *Data, err error) // Closure for refresh/revalidation

	refCount int64 // refCount for concurrent/lifecycle management
	isDoomed int64 // "Doomed" flag for objects marked for delete but still referenced

	beta               int64 // e.g. 0.7*100 = 70, parameter for probabilistic refresh
	minStaleDuration   int64 // How long to keep without any refresh (nanoseconds)
	revalidateInterval int64 // Interval for background revalidation (nanoseconds)
	revalidatedAt      int64 // Last revalidated time (nanoseconds since epoch)
}

// NewResponse constructs a new Response using memory pools and sets up all fields.
func NewResponse(
	data *Data, req *Request, cfg *config.Cache,
	revalidator func(ctx context.Context) (data *Data, err error),
) (*Response, error) {
	return ResponsePool.Get().Init().clear().SetUp(
		cfg.RevalidateBeta, cfg.RevalidateInterval,
		cfg.RefreshDurationThreshold, data, req, revalidator,
	), nil
}

// Init ensures all pointers are non-nil after pool Get.
func (r *Response) Init() *Response {
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

// SetUp stores the Data, Request, and config-driven fields into the Response.
func (r *Response) SetUp(beta float64, interval, minStale time.Duration, data *Data, req *Request, revalidator func(ctx context.Context) (data *Data, err error)) *Response {
	r.data.Store(data)
	r.request.Store(req)
	r.revalidator = revalidator
	r.revalidatedAt = time.Now().UnixNano()
	r.beta = int64(beta * 100)
	r.revalidateInterval = interval.Nanoseconds()
	r.minStaleDuration = minStale.Nanoseconds()
	return r
}

// clear resets the Response for re-use from the pool.
func (r *Response) clear() *Response {
	r.revalidator = nil
	r.refCount = 0
	r.isDoomed = 0
	r.revalidatedAt = 0
	r.minStaleDuration = 0
	r.revalidateInterval = 0
	r.beta = 0
	r.lruListElem.Store(nil)
	r.request.Store(nil)
	r.data.Store(nil)
	return r
}

// --- Response API ---

func (r *Response) Touch() *Response {
	atomic.StoreInt64(&r.revalidatedAt, time.Now().UnixNano())
	return r
}

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

	var (
		beta             = atomic.LoadInt64(&r.beta)
		interval         = atomic.LoadInt64(&r.revalidateInterval)
		revalidatedAt    = atomic.LoadInt64(&r.revalidatedAt)
		minStaleDuration = atomic.LoadInt64(&r.minStaleDuration)
	)

	if r.data.Load().statusCode != http.StatusOK {
		interval = interval / 10                 // On stale will be used 10% of origin interval.
		minStaleDuration = minStaleDuration / 10 // On stale will be used 10% of origin stale duration.
	}

	// hard check that min
	if age := time.Since(time.Unix(0, revalidatedAt)).Nanoseconds(); age > minStaleDuration {
		return rand.Float64() >= math.Exp((float64(-beta)/100)*float64(age)/float64(interval))
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
	ResponsePool.Put(r)
	return true
}

// MarshalBinary serializes Response+Request+Data into a length-prefixed binary format.
// NO string/reflect, только "сырье"!
func (r *Response) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	// ---- Request ----
	req := r.Request()
	if req == nil {
		return nil, errors.New("nil request")
	}
	// Write project, domain, language
	for _, field := range [][]byte{req.project, req.domain, req.language} {
		if err := writeBytes(&buf, field); err != nil {
			return nil, err
		}
	}
	// Write tags count
	if err := binary.Write(&buf, binary.LittleEndian, uint8(len(req.tags))); err != nil {
		return nil, err
	}
	for _, tag := range req.tags {
		if err := writeBytes(&buf, tag); err != nil {
			return nil, err
		}
	}
	// Write key, shardKey
	if err := binary.Write(&buf, binary.LittleEndian, req.key); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, req.shardKey); err != nil {
		return nil, err
	}

	// ---- Data ----
	data := r.Data()
	if data == nil {
		return nil, errors.New("nil data")
	}
	// Status code
	if err := binary.Write(&buf, binary.LittleEndian, int32(data.statusCode)); err != nil {
		return nil, err
	}
	// Headers (map[string][]string)
	if err := writeHeaders(&buf, data.headers); err != nil {
		return nil, err
	}
	// Body
	if err := writeBytes(&buf, data.body); err != nil {
		return nil, err
	}

	// ---- Meta ----
	// beta, minStaleDuration, revalidateInterval, revalidatedAt
	meta := []int64{r.beta, r.minStaleDuration, r.revalidateInterval, r.revalidatedAt}
	for _, v := range meta {
		if err := binary.Write(&buf, binary.LittleEndian, v); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func writeBytes(w io.Writer, b []byte) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(b))); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

func writeHeaders(w io.Writer, hdr map[string][]string) error {
	// Write headers count
	if err := binary.Write(w, binary.LittleEndian, uint32(len(hdr))); err != nil {
		return err
	}
	for k, vals := range hdr {
		if err := writeBytes(w, []byte(k)); err != nil {
			return err
		}
		// Write count of values
		if err := binary.Write(w, binary.LittleEndian, uint32(len(vals))); err != nil {
			return err
		}
		for _, v := range vals {
			if err := writeBytes(w, []byte(v)); err != nil {
				return err
			}
		}
	}
	return nil
}

// UnmarshalBinary deserializes Response+Request+Data from length-prefixed binary format.
func (r *Response) UnmarshalBinary(data []byte, revalidatorMaker func(req *Request) func(ctx context.Context) (*Data, error)) error {
	buf := bytes.NewReader(data)

	// ---- Request ----
	project, err := readBytes(buf)
	if err != nil {
		return err
	}
	domain, err := readBytes(buf)
	if err != nil {
		return err
	}
	language, err := readBytes(buf)
	if err != nil {
		return err
	}

	var tagsCount uint8
	if err = binary.Read(buf, binary.LittleEndian, &tagsCount); err != nil {
		return err
	}
	tags := tagsSlicesPool.Get()
	for i := 0; i < int(tagsCount); i++ {
		tag, err := readBytes(buf)
		if err != nil {
			return err
		}
		tags[i] = internSlice(tag)
	}
	tags = tags[:tagsCount]

	var key, shardKey uint64
	if err = binary.Read(buf, binary.LittleEndian, &key); err != nil {
		return err
	}
	if err = binary.Read(buf, binary.LittleEndian, &shardKey); err != nil {
		return err
	}

	// ---- Data ----
	var statusCode int32
	if err = binary.Read(buf, binary.LittleEndian, &statusCode); err != nil {
		return err
	}

	headers, err := readHeaders(buf)
	if err != nil {
		return err
	}

	body, err := readBytes(buf)
	if err != nil {
		return err
	}

	// ---- Meta ----
	meta := make([]int64, 4)
	for i := range meta {
		if err = binary.Read(buf, binary.LittleEndian, &meta[i]); err != nil {
			return err
		}
	}

	// --- Make objects ---
	// Data
	dataObj := dataPool.Get()
	*dataObj = Data{
		statusCode: int(statusCode),
		headers:    headers,
		body:       body,
		releaseFn:  func() {}, // no release
	}

	// Request
	reqObj := requestsPool.Get().clear().setUp(
		project, domain, language, tags, func() { tagsSlicesPool.Put(tags) },
	)

	// Response
	r.clear().SetUp(
		float64(meta[0])/100,   // beta
		time.Duration(meta[2]), // revalidateInterval
		time.Duration(meta[1]), // minStaleDuration
		dataObj,
		reqObj,
		revalidatorMaker(reqObj),
	)

	return nil
}

// readBytes reads []byte in format [len:uint32][data...]
func readBytes(r io.Reader) ([]byte, error) {
	var l uint32
	if err := binary.Read(r, binary.LittleEndian, &l); err != nil {
		return nil, err
	}
	if l == 0 {
		return nil, nil
	}
	b := make([]byte, l)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

// readHeaders reads map[string][]string from binary format.
func readHeaders(r io.Reader) (http.Header, error) {
	var count uint32
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	hdr := make(http.Header, int(count))
	for i := 0; i < int(count); i++ {
		k, err := readBytes(r)
		if err != nil {
			return nil, err
		}
		var valsCount uint32
		if err := binary.Read(r, binary.LittleEndian, &valsCount); err != nil {
			return nil, err
		}
		for j := 0; j < int(valsCount); j++ {
			v, err := readBytes(r)
			if err != nil {
				return nil, err
			}
			hdr.Add(string(k), string(v))
		}
	}
	return hdr, nil
}
