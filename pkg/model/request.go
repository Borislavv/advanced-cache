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
	preallocatedBufferCapacity = 256 // For pre-allocated buffer for query strings and URL
	tagBufferCapacity          = 128 // For each tag's []byte slice
	maxTagsLen                 = 10  // Maximum number of tags per request
)

// internPool maintains strict string interning for high efficiency in memory usage.
// All unique byte slices (e.g. project/domain/tag names) are pooled here for deduplication.
type internPool struct {
	mu *sync.RWMutex
	mm map[string][]byte
}

var (
	// interningPool stores unique byte slices to enforce strict interning and avoid duplication.
	interningPool = &internPool{
		mu: &sync.RWMutex{},
		mm: make(map[string][]byte, synced.PreallocationBatchSize*10),
	}

	// Pools for reusable objects and buffers.
	hasherPool = synced.NewBatchPool[*xxh3.Hasher](synced.PreallocationBatchSize, func() *xxh3.Hasher {
		return xxh3.New()
	})
	requestsPool = synced.NewBatchPool[*Request](synced.PreallocationBatchSize, func() *Request {
		return &Request{
			uniqueQuery: make([]byte, 0, preallocatedBufferCapacity+(maxTagsLen+tagBufferCapacity)),
		}
	})
	keyBufferPool = synced.NewBatchPool[[]byte](synced.PreallocationBatchSize, func() []byte {
		return make([]byte, 0, preallocatedBufferCapacity)
	})
	// Used for one-time preallocation; not reused as a typical pool.
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

// internSlice returns a shared []byte for identical inputs to reduce allocations (true string interning).
func internSlice(b []byte) []byte {
	key := string(b)

	interningPool.mu.RLock()
	if v, ok := interningPool.mm[key]; ok {
		interningPool.mu.RUnlock()
		return v
	}
	interningPool.mu.RUnlock()

	slBytes := preallocatorBufferPool.Get() // "Plays" as a preallocator, not a true pool.
	slBytes = slBytes[:0]
	slBytes = append(slBytes, b...)

	interningPool.mu.Lock()
	interningPool.mm[key] = slBytes
	interningPool.mu.Unlock()

	return slBytes
}

// Request holds normalized, deduplicated, hashed and uniquely queryable representation of a cache request.
type Request struct {
	project     []byte   // Interned project name
	domain      []byte   // Interned domain
	language    []byte   // Interned language
	tags        [][]byte // Array of interned tag values
	uniqueQuery []byte   // Built query string (for HTTP requests)
	key         uint64   // xxh3 hash of all above fields (acts as the main cache key)
	shardKey    uint64   // Which shard this request maps to
	releaseFn   func()   // Function to release underlying resources/buffers
}

// NewManualRequest creates a Request with explicit parameters, bypassing fasthttp.
func NewManualRequest(project, domain, language []byte, tags [][]byte) (*Request, error) {
	r, err := requestsPool.Get().clear().setUp(project, domain, language, tags, func() {}).validate()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// NewRequest builds a Request from fasthttp.Args, with strict interning and pooling.
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

// setUp initializes the Request, interns all fields, builds keys, and sets up uniqueQuery.
func (r *Request) setUp(project, domain, language []byte, tags [][]byte, releaseFn func()) *Request {
	// Strict interning: all input slices are deduped
	r.project = internSlice(project)
	r.domain = internSlice(domain)
	r.language = internSlice(language)

	// Intern each tag value
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

// clear resets the Request to zero (except for buffer capacity).
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

// validate ensures that required fields are set (project, domain, language).
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

// extractTags collects choice-like tags from the args, returns them and a release function.
func extractTags(args *fasthttp.Args) ([][]byte, func()) {
	var (
		nullValue   = []byte("null")
		choiceValue = []byte("choice")
	)

	tagsSls := tagsSlicesPool.Get()
	// Reset each tag slice
	for i := range tagsSls {
		tagsSls[i] = tagsSls[i][:0]
	}

	i := 0
	args.VisitAll(func(key, value []byte) {
		if !bytes.HasPrefix(key, choiceValue) || bytes.Equal(value, nullValue) {
			return
		}
		tagsSls[i] = internSlice(value)
		i++
	})

	// Only return used tags
	tags := tagsSls[:i]
	return tags, func() { tagsSlicesPool.Put(tagsSls) }
}

// Getters for all important fields.
func (r *Request) GetProject() []byte  { return r.project }
func (r *Request) GetDomain() []byte   { return r.domain }
func (r *Request) GetLanguage() []byte { return r.language }
func (r *Request) GetTags() [][]byte   { return r.tags }

// setUpKey computes a unique hash key (xxh3) for this request.
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

// Key returns the computed hash key for the request.
func (r *Request) Key() uint64 { return r.key }

// ShardKey returns the precomputed shard index.
func (r *Request) ShardKey() uint64 { return r.shardKey }

// setUpShardKey computes the shard index from the key.
func (r *Request) setUpShardKey(key uint64) {
	r.shardKey = sharded.MapShardKey(key)
}

// setUpQuery builds the query string for backend usage (uses same buffer for efficiency).
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

// ToQuery returns the generated query string.
func (r *Request) ToQuery() []byte { return r.uniqueQuery }

// Release releases and resets the request (and any underlying buffer/tag slices) for reuse.
func (r *Request) Release() {
	r.clear()
	r.releaseFn()
	requestsPool.Put(r)
}
