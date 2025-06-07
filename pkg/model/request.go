package model

import (
	"bytes"
	"errors"
	"sync"
	"sync/atomic"

	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/xxh3"
)

const (
	preallocatedBufferCapacity = 256
	tagBufferCapacity          = 128
	maxTagsLen                 = 10
)

var (
	hasherPool = synced.NewBatchPool[*xxh3.Hasher](synced.PreallocationBatchSize, func() *xxh3.Hasher {
		return xxh3.New()
	})
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

var internPool sync.Map

func internBytes(b []byte) []byte {
	strKey := string(b)
	if val, ok := internPool.Load(strKey); ok {
		return val.([]byte)
	}
	cp := append([]byte(nil), b...) // делаем копию
	actual, _ := internPool.LoadOrStore(strKey, cp)
	return actual.([]byte)
}

type InternedBytes struct {
	data []byte
}

func (k InternedBytes) Equal(other InternedBytes) bool {
	return bytes.Equal(k.data, other.data)
}

func (k InternedBytes) Hash() uint64 {
	return xxh3.Hash(k.data)
}

func (k InternedBytes) String() string {
	return string(k.data)
}

func (k InternedBytes) MarshalBinary() ([]byte, error) {
	return k.data, nil
}

func (k InternedBytes) UnmarshalBinary(b []byte) error {
	k.data = append(k.data[:0], b...)
	return nil
}

func (k InternedBytes) GoString() string {
	return string(k.data)
}

type Request struct {
	mu         *sync.RWMutex
	project    []byte
	domain     []byte
	language   []byte
	tags       [][]byte
	freeTagsFn FreeResourceFunc

	uniqueQuery []byte
	uniqueKey   *atomic.Uint64
}

func NewRequest(q *fasthttp.Args) (*Request, error) {
	tags, freeTagsFn := extractTags(q)
	return NewManualRequest(q.Peek("project[id]"), q.Peek("domain"), q.Peek("language"), tags, freeTagsFn)
}

func NewManualRequest(project, domain, language []byte, tags [][]byte, freeTagsFn FreeResourceFunc) (*Request, error) {
	if len(project) == 0 {
		return nil, errors.New("project is not specified")
	}
	if len(domain) == 0 {
		return nil, errors.New("domain is not specified")
	}
	if len(language) == 0 {
		return nil, errors.New("language is not specified")
	}

	req := RequestsPool.Get()
	*req = Request{
		mu:        &sync.RWMutex{},
		uniqueKey: &atomic.Uint64{},
	}

	req.project = internBytes(project)
	req.domain = internBytes(domain)
	req.language = internBytes(language)

	for i, tag := range tags {
		if len(tag) > 0 {
			tags[i] = internBytes(tag)
		}
	}
	req.tags = tags
	req.freeTagsFn = freeTagsFn

	return req, nil
}

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

	bufCap := len(r.project) + len(r.domain) + len(r.language)
	for _, tag := range r.tags {
		bufCap += len(tag)
	}

	buf := make([]byte, 0, bufCap)
	buf = append(buf, r.project...)
	buf = append(buf, r.domain...)
	buf = append(buf, r.language...)
	for _, tag := range r.tags {
		buf = append(buf, tag...)
	}

	hasher := hasherPool.Get()
	defer hasherPool.Put(hasher)
	hasher.Reset()
	_, _ = hasher.Write(buf)

	key := hasher.Sum64()
	r.uniqueKey.Store(key)
	return key
}

func (r *Request) ToQuery() []byte {
	r.mu.RLock()
	if len(r.uniqueQuery) > 0 {
		defer r.mu.RUnlock()
		return r.uniqueQuery
	}
	r.mu.RUnlock()

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
	RequestsPool.Put(r)
}
