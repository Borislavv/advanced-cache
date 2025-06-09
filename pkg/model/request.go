package model

import (
	"bytes"
	"errors"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/xxh3"
	"sync/atomic"
)

const (
	preallocatedBufferCapacity = 256 // for pre-allocated buffer for concat string literals and url
	tagBufferCapacity          = 128
	maxTagsLen                 = 10
)

var (
	hasherPool = synced.NewBatchPool[*xxh3.Hasher](synced.PreallocationBatchSize, func() *xxh3.Hasher {
		return xxh3.New()
	})
	requestsPool = synced.NewBatchPool[*Request](synced.PreallocationBatchSize, func() *Request {
		r := &Request{
			project:     nil,
			domain:      nil,
			language:    nil,
			tags:        nil,
			releaseFn:   nil,
			uniqueQuery: make([]byte, 0, preallocatedBufferCapacity+(maxTagsLen+tagBufferCapacity)),
			key:         0,
			shardKey:    0,
		}
		return r
	})
	keyBufferPool = synced.NewBatchPool[[]byte](synced.PreallocationBatchSize, func() []byte {
		return make([]byte, 0, preallocatedBufferCapacity)
	})
	tagsSlicesPool = synced.NewBatchPool[[][]byte](synced.PreallocationBatchSize, func() [][]byte {
		batch := make([][]byte, maxTagsLen)
		for i := range batch {
			batch[i] = make([]byte, 0, tagBufferCapacity)
		}
		return batch
	})
)

type Request struct {
	project     []byte
	domain      []byte
	language    []byte
	tags        [][]byte
	uniqueQuery []byte
	key         uint64
	shardKey    uint64
	releaseFn   func()
}

func NewManualRequest(project []byte, domain []byte, language []byte, tags [][]byte) (*Request, error) {
	req, err := requestsPool.Get().clear().setUp(project, domain, language, tags, func() {}).validate()
	if err != nil {
		return nil, err
	}
	return req, nil
}

func NewRequest(q *fasthttp.Args) (*Request, error) {
	var (
		project         = q.Peek("project[id]")
		domain          = q.Peek("domain")
		language        = q.Peek("language")
		tags, releaseFn = extractTags(q)
	)
	req, err := requestsPool.Get().clear().setUp(project, domain, language, tags, releaseFn).validate()
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (r *Request) setUp(project, domain, language []byte, tags [][]byte, releaseFn func()) *Request {
	r.project = project
	r.domain = domain
	r.language = language
	r.tags = tags
	r.releaseFn = releaseFn
	r.uniqueQuery = r.uniqueQuery[:0]

	key := r.setUpKey()
	r.setUpQuery()
	r.setUpShardKey(key)

	return r
}

func (r *Request) clear() *Request {
	r.project = nil
	r.domain = nil
	r.language = nil
	r.tags = nil
	r.key = 0
	r.shardKey = 0
	r.releaseFn = nil
	return r
}

func (r *Request) validate() (*Request, error) {
	if r.project == nil || len(r.project) == 0 {
		return nil, errors.New("project is not specified")
	}
	if r.domain == nil || len(r.domain) == 0 {
		return nil, errors.New("domain is not specified")
	}
	if r.language == nil || len(r.language) == 0 {
		return nil, errors.New("language is not specified")
	}
	return r, nil
}

// extractTags - returns a slice with []byte("${choice name}").
func extractTags(args *fasthttp.Args) (tags [][]byte, releaseFn func()) {
	var (
		nullValue   = []byte("null")
		choiceValue = []byte("choice")
	)

	tagsSls := tagsSlicesPool.Get()
	for _, tagSl := range tagsSls {
		tagSl = tagSl[:0]
	}

	i := 0
	args.VisitAll(func(key, value []byte) {
		if !bytes.HasPrefix(key, choiceValue) || bytes.Equal(value, nullValue) {
			return
		}
		tagsSls[i] = append(tagsSls[i], value...)
		i++
	})
	return tagsSls, func() { tagsSlicesPool.Put(tagsSls) }
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
func (r *Request) setUpKey() uint64 {
	// build unique buffer of request data
	buf := keyBufferPool.Get()
	defer keyBufferPool.Put(buf)
	buf = buf[:0]

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
	r.key = key

	return key
}
func (r *Request) Key() uint64 {
	return r.key
}
func (r *Request) ShardKey() uint64 {
	return atomic.LoadUint64(&r.shardKey)
}

func (r *Request) setUpShardKey(key uint64) {
	atomic.StoreUint64(&r.shardKey, sharded.MapShardKey(key))
}

func (r *Request) setUpQuery() {
	q := r.uniqueQuery
	if q == nil {
		panic("query slice must be set and be alive along all request live")
	}

	buf := q[:0]
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

	r.uniqueQuery = buf
}

func (r *Request) ToQuery() []byte {
	return r.uniqueQuery
}

func (r *Request) Release() {
	r.clear().releaseFn()
	requestsPool.Put(r)
}
