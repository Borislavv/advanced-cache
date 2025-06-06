package utils

import (
	"bytes"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"net/http"
)

var PagedataResponsesPool = synced.NewBatchPool[*bytes.Buffer](1000, func() *bytes.Buffer {
	return new(bytes.Buffer)
})

func ReadResponseBody(resp *http.Response) ([]byte, error) {
	buf := PagedataResponsesPool.Get()
	defer func() {
		buf.Reset()
		PagedataResponsesPool.Put(buf)
	}()

	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes()[:], nil
}
