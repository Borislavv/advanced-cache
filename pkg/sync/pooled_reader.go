package synced

import (
	"bytes"
	"net/http"
)

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
	pool *BatchPool[*bytes.Buffer]
}

// NewPooledResponseReader creates a new PooledResponseReader with a given initial buffer pool size.
func NewPooledResponseReader(preallocatedBuffers int) *PooledResponseReader {
	return &PooledResponseReader{
		pool: NewBatchPool[*bytes.Buffer](preallocatedBuffers, func() *bytes.Buffer {
			return new(bytes.Buffer)
		}),
	}
}

// Read reads the HTTP response body into a pooled buffer and returns the bytes.
// IMPORTANT: The returned bode slice aliases a pooled buffer and must not be used after free() is called.
// Data race or use-after-free is possible if bode is accessed after calling free().
func (r *PooledResponseReader) Read(resp *http.Response) (bode []byte, free FreeResourceFunc, err error) {
	buf := r.pool.Get()
	buf.Reset() // Defensive: ensure buffer is empty
	freeFn := func() { r.pool.Put(buf) }
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, freeFn, err
	}
	return buf.Bytes(), freeFn, nil
}
