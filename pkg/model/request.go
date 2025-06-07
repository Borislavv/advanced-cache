package model

import (
	"bytes"
	"errors"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/xxh3"
	"go.uber.org/atomic"
	"sync"
)

const (
	preallocatedBufferCapacity = 256 // for pre-allocated buffer for concat string literals and url
	tagBufferCapacity          = 128
	maxTagsLen                 = 10
)

var hasherPool = synced.NewBatchPool[*xxh3.Hasher](synced.PreallocationBatchSize, func() *xxh3.Hasher {
	return xxh3.New()
})

var (
	RequestsPool = synced.NewBatchPool[*Request](synced.PreallocationBatchSize, func() *Request {
		return &Request{}
	})
	TagsSlicesPool = synced.NewBatchPool[[][]byte](synced.PreallocationBatchSize, func() [][]byte {
		batch := make([][]byte, maxTagsLen)
		for i := range batch {
			batch[i] = make([]byte, 0, tagBufferCapacity)
		}
		return batch
	})
)

type FreeResourceFunc func()

type Request struct {
	mu         *sync.RWMutex
	project    []byte
	domain     []byte
	language   []byte
	tags       [][]byte
	freeTagsFn FreeResourceFunc

	// calculated fields (calling while init.)
	uniqueQuery []byte
	uniqueKey   *atomic.Uint64
}

func NewRequest(q *fasthttp.Args) (*Request, error) {
	tags, freeTagsFn := extractTags(q)
	return NewManualRequest(q.Peek("project[id]"), q.Peek("domain"), q.Peek("language"), tags, freeTagsFn)
}

func NewManualRequest(project, domain, language []byte, tags [][]byte, freeTagsFn FreeResourceFunc) (*Request, error) {
	if project == nil || len(project) == 0 {
		return nil, errors.New("project is not specified")
	}
	if domain == nil || len(domain) == 0 {
		return nil, errors.New("domain is not specified")
	}
	if language == nil || len(language) == 0 {
		return nil, errors.New("language is not specified")
	}

	req := RequestsPool.Get()
	*req = Request{
		mu:        &sync.RWMutex{},
		uniqueKey: &atomic.Uint64{},
	}

	if req.mu == nil {
		req.mu = new(sync.RWMutex)
	}
	if req.uniqueKey == nil {
		req.uniqueKey = atomic.NewUint64(0)
	}

	req.project = project
	req.domain = domain
	req.language = language
	req.tags = tags
	req.freeTagsFn = freeTagsFn

	return req, nil
}

// extractTags - returns a slice with []byte("${choice name}").
func extractTags(args *fasthttp.Args) ([][]byte, FreeResourceFunc) {
	var (
		nullValue   = []byte("null")
		choiceValue = []byte("choice")
	)

	tagsSls := TagsSlicesPool.Get()
	for i := range tagsSls {
		tagsSls[i] = tagsSls[i][:0]
	}

	idx := 0
	args.VisitAll(func(key, value []byte) {
		if !bytes.HasPrefix(key, choiceValue) || bytes.Equal(value, nullValue) || idx >= maxTagsLen {
			return
		}
		tagsSls[idx] = append(tagsSls[idx], value...)
		idx++
	})

	return tagsSls[:idx], func() { TagsSlicesPool.Put(tagsSls) }
}

func (r *Request) GetProject() []byte {
	return r.project
}
func (r *Request) GetDomain() []byte {
	return r.domain
}
func (r *Request) GetLanguage() []byte {
	return r.language
}
func (r *Request) GetTags() [][]byte {
	return r.tags
}
func (r *Request) UniqueKey() uint64 {
	if key := r.uniqueKey.Load(); key != 0 {
		return key
	}

	// calculate capacity
	bufCap := len(r.project) + len(r.domain) + len(r.language)
	for _, tag := range r.tags {
		bufCap += len(tag)
	}

	// build unique buffer of request data
	buf := make([]byte, 0, bufCap)
	buf = append(buf, r.project...)
	buf = append(buf, r.domain...)
	buf = append(buf, r.language...)
	for _, tag := range r.tags {
		if len(tag) == 0 {
			continue
		}
		buf = append(buf, tag...)
	}

	// calculate hash of unique []byte
	hasher := hasherPool.Get()
	defer hasherPool.Put(hasher)
	hasher.Reset()
	if _, err := hasher.Write(buf); err != nil {
		panic(err)
	}

	key := hasher.Sum64()
	r.uniqueKey.Store(key)

	return key
}
func (r *Request) ToQuery() []byte {
	r.mu.RLock()
	uniqueQuery := r.uniqueQuery
	r.mu.RUnlock()

	if len(uniqueQuery) != 0 {
		return uniqueQuery
	}

	bufCap := preallocatedBufferCapacity + len(r.project) + len(r.domain) + len(r.language)
	for _, tag := range r.tags {
		bufCap += len(tag)
	}

	buf := make([]byte, 0, bufCap)
	buf = append(buf, []byte("?project[id]=")...)
	buf = append(buf, r.project...)
	buf = append(buf, []byte("&domain=")...)
	buf = append(buf, r.domain...)
	buf = append(buf, []byte("&language=")...)
	buf = append(buf, r.language...)

	var n int
	for _, tag := range r.tags {
		if len(tag) == 0 {
			continue
		}
		buf = append(buf, []byte("&choice")...)
		buf = append(buf, bytes.Repeat([]byte("[choice]"), n)...)
		buf = append(buf, []byte("[name]=")...)
		buf = append(buf, tag...)
		n++
	}
	buf = append(buf, []byte("&choice")...)
	buf = append(buf, bytes.Repeat([]byte("[choice]"), n)...)
	buf = append(buf, []byte("=null")...)

	r.mu.Lock()
	r.uniqueQuery = buf
	r.mu.Unlock()

	return buf
}

func (r *Request) Free() {
	r.freeTagsFn()
	r.uniqueKey.Store(0)
	if cap(r.uniqueQuery) > 0 {
		r.uniqueQuery = r.uniqueQuery[:0]
	}
}
