package model

import (
	"bytes"
	"errors"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/xxh3"
	"sync"
)

const (
	preallocatedBufferCapacity = 256 // for pre-allocated buffer for concat string literals and url
	tagBufferCapacity          = 128
	maxTagsLen                 = 10
)

type internPool struct {
	mu *sync.RWMutex
	mm map[string][]byte
}

var (
	// internPool stores unique byte slices to enforce strict interning
	interningPool = &internPool{
		mu: &sync.RWMutex{},
		mm: make(map[string][]byte, synced.PreallocationBatchSize*10),
	}

	hasherPool = synced.NewBatchPool[*xxh3.Hasher](synced.PreallocationBatchSize, func() *xxh3.Hasher {
		return xxh3.New()
	})
	requestsPool = synced.NewBatchPool[*Request](synced.PreallocationBatchSize, func() *Request {
		r := &Request{
			uniqueQuery: make([]byte, 0, preallocatedBufferCapacity+(maxTagsLen+tagBufferCapacity)),
		}
		return r
	})
	keyBufferPool = synced.NewBatchPool[[]byte](synced.PreallocationBatchSize, func() []byte {
		return make([]byte, 0, preallocatedBufferCapacity)
	})
	// All this buffers will be stored in sync.Map as interned bytes for some keys. It's mean that this pool is more preallocator than pool.
	preallocatorBufferPool = synced.NewBatchPool[[]byte](synced.PreallocationBatchSize, func() []byte {
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

// internSlice returns a shared []byte for identical inputs to reduce allocations.
func internSlice(b []byte) []byte {
	key := string(b)

	interningPool.mu.RLock()
	if v, ok := interningPool.mm[key]; ok {
		interningPool.mu.RUnlock()
		return v
	}
	interningPool.mu.RUnlock()

	// plays as a preallocator
	slBytes := preallocatorBufferPool.Get()
	slBytes = slBytes[:0]
	slBytes = append(slBytes, b...)

	interningPool.mu.Lock()
	interningPool.mm[key] = slBytes
	interningPool.mu.Unlock()

	return slBytes
}

// Request holds cached request context data.
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

// NewManualRequest creates a Request with explicit parameters.
func NewManualRequest(project, domain, language []byte, tags [][]byte) (*Request, error) {
	r, err := requestsPool.Get().clear().setUp(project, domain, language, tags, func() {}).validate()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// NewRequest builds a Request from fasthttp.Args.
func NewRequest(q *fasthttp.Args) (*Request, error) {
	var (
		project         = q.Peek("project[id]")
		domain          = q.Peek("domain")
		language        = q.Peek("language")
		tags, releaseFn = extractTags(q)
	)
	r, err := requestsPool.Get().clear().setUp(project, domain, language, tags, releaseFn).validate()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// setUp initializes the Request fields, interns byte slices and prepares keys.
func (r *Request) setUp(project, domain, language []byte, tags [][]byte, releaseFn func()) *Request {
	// strict interning of input slices
	r.project = internSlice(project)
	r.domain = internSlice(domain)
	r.language = internSlice(language)

	// intern each tag value
	for i, tag := range tags {
		if len(tag) > 0 {
			tags[i] = internSlice(tag)
		}
	}
	r.tags = tags
	r.releaseFn = releaseFn
	r.uniqueQuery = r.uniqueQuery[:0]

	r.setUpQuery()
	r.setUpShardKey(r.setUpKey())

	return r
}

// clear resets Request to initial state (except buffer capacities).
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

// validate ensures mandatory fields are set.
func (r *Request) validate() (*Request, error) {
	if len(r.project) == 0 {
		return nil, errors.New("project is not specified")
	}
	if len(r.domain) == 0 {
		return nil, errors.New("domain is not specified")
	}
	if len(r.language) == 0 {
		return nil, errors.New("language is not specified")
	}
	return r, nil
}

// extractTags collects tag values and returns a release function.
func extractTags(args *fasthttp.Args) ([][]byte, func()) {
	var (
		nullValue   = []byte("null")
		choiceValue = []byte("choice")
	)

	tagsSls := tagsSlicesPool.Get()
	// correctly reset each slice
	for i := range tagsSls {
		tagsSls[i] = tagsSls[i][:0]
	}

	i := 0
	args.VisitAll(func(key, value []byte) {
		if !bytes.HasPrefix(key, choiceValue) || bytes.Equal(value, nullValue) {
			return
		}
		// append interned tag value
		tagsSls[i] = internSlice(value)
		i++
	})

	// return only actual tags
	tags := tagsSls[:i]
	return tags, func() { tagsSlicesPool.Put(tagsSls) }
}

// Accessors
func (r *Request) GetProject() []byte  { return r.project }
func (r *Request) GetDomain() []byte   { return r.domain }
func (r *Request) GetLanguage() []byte { return r.language }
func (r *Request) GetTags() [][]byte   { return r.tags }

// setUpKey computes and stores the unique key for the request.
func (r *Request) setUpKey() uint64 {
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

// Key returns the computed hash key.
func (r *Request) Key() uint64 { return r.key }

// ShardKey returns the computed shard key.
func (r *Request) ShardKey() uint64 { return r.shardKey }

// setUpShardKey computes the shard based on the key.
func (r *Request) setUpShardKey(key uint64) {
	r.shardKey = sharded.MapShardKey(key)
}

// setUpQuery builds the query string buffer.
func (r *Request) setUpQuery() {
	if r.uniqueQuery == nil {
		panic("query slice must be set and be alive along all request live")
	}

	buf := r.uniqueQuery[:0]
	buf = append(buf, []byte("?project[id]=")...)
	buf = append(buf, r.project...)
	buf = append(buf, []byte("&domain=")...)
	buf = append(buf, r.domain...)
	buf = append(buf, []byte("&language=")...)
	buf = append(buf, r.language...)

	n := 0
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

// ToQuery returns the built query.
func (r *Request) ToQuery() []byte { return r.uniqueQuery }

// Release resets and returns the Request to the pool.
func (r *Request) Release() {
	r.clear()
	r.releaseFn()
	requestsPool.Put(r)
}
