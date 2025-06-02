package utils

import (
	"bytes"
	"net/http"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func ReadResponseBody(resp *http.Response) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return append([]byte(nil), buf.Bytes()...), nil
}
