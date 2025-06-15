package synced

import (
	"bytes"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/types"
	"net/http"
	"unsafe"
)

var ResponseReaderBufferPool = NewBatchPool[*types.SizedBox[*bytes.Buffer]](PreallocateBatchSize, func() *types.SizedBox[*bytes.Buffer] {
	return &types.SizedBox[*bytes.Buffer]{
		Value: new(bytes.Buffer),
		CalcWeightFn: func(s *types.SizedBox[*bytes.Buffer]) int64 {
			return int64(unsafe.Sizeof(*s)) + int64(s.Value.Cap())
		},
	}
})

// FreeResourceFunc is a function to release/recycle a pooled resource after use.
type FreeResourceFunc func()

// PooledReader is an interface for reading HTTP responses into pooled byte slices (with explicit free).
type PooledReader interface {
	// Read reads the response body into a pooled buffer, returning:
	// - bode:   byte slice with the entire response body (may alias an internal buffer)
	// - free:   function to call when you are done with bode (to return buffer to pool)
	// - err:    any error from reading
	Read(resp *http.Response) (bode []byte, free FreeResourceFunc, err error)
}

// PooledResponseReader reads HTTP response bodies into pooled buffers for reuse,
// minimizing allocations under high load. **Not thread-safe for returned body after free!**
type PooledResponseReader struct {
}

// NewPooledResponseReader creates a new PooledResponseReader with a given initial buffer pool size.
func NewPooledResponseReader(preallocatedBuffers int) *PooledResponseReader {
	return &PooledResponseReader{}
}

// Read reads the HTTP response body into a pooled buffer and returns the bytes.
// IMPORTANT: The returned bode slice aliases a pooled buffer and must not be used after free() is called.
// Data race or use-after-free is possible if bode is accessed after calling free().
func (r *PooledResponseReader) Read(resp *http.Response) (bode []byte, free FreeResourceFunc, err error) {
	buf := ResponseReaderBufferPool.Get()
	buf.Value.Reset() // Defensive: ensure buffer is empty
	freeFn := func() { ResponseReaderBufferPool.Put(buf) }
	_, err = buf.Value.ReadFrom(resp.Body)
	if err != nil {
		return nil, freeFn, err
	}
	return buf.Value.Bytes(), freeFn, nil
}
