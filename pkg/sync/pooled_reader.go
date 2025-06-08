package synced

import (
	"bytes"
	"net/http"
)

type FreeResourceFunc func()

type PooledReader interface {
	Read(resp *http.Response) (bode []byte, free FreeResourceFunc, err error)
}

type PooledResponseReader struct {
	pool *BatchPool[*bytes.Buffer]
}

func NewPooledResponseReader(preallocatedBuffers int) *PooledResponseReader {
	return &PooledResponseReader{
		pool: NewBatchPool[*bytes.Buffer](preallocatedBuffers, func() *bytes.Buffer {
			return new(bytes.Buffer)
		}),
	}
}

/**
 * IMPORTANT! UNSAFE! If bode will alive and still in use after freed, possible data race and dangling pointers.
 *
 * Reads response body into a buffer from sync.Pool and returns this buffer when you call "free" response value.
 */
func (r *PooledResponseReader) Read(resp *http.Response) (bode []byte, free FreeResourceFunc, err error) {
	//buf := r.pool.Get()
	//buf.Reset()
	var buf bytes.Buffer
	//freeFn := func() { r.pool.Put(buf) }
	freeFn := func() {}
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, freeFn, err
	}
	return buf.Bytes()[:], freeFn, nil
}
