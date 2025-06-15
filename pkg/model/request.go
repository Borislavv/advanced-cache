package model

import (
	"bytes"
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/consts"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/types"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/xxh3"
	"sync/atomic"
	"unsafe"
)

const (
	preallocatedBufferCapacity = 256 // For pre-allocated buffer for query strings and URL
	tagBufferCapacity          = 128 // For each tag's []byte slice
	maxTagsLen                 = 10  // Maximum number of tags per request
)

var (
	// Pools for reusable objects and buffers.
	HasherPool = synced.NewBatchPool[*types.SizedBox[*xxh3.Hasher]](synced.PreallocateBatchSize, func() *types.SizedBox[*xxh3.Hasher] {
		return &types.SizedBox[*xxh3.Hasher]{
			Value: xxh3.New(),
			CalcWeightFn: func(box *types.SizedBox[*xxh3.Hasher]) int64 {
				return int64(unsafe.Sizeof(*box.Value)) + 1024 + 64 + 8 // digged inside struct and calculated
			},
		}
	})
	RequestsPool = synced.NewBatchPool[*Request](synced.PreallocateBatchSize, func() *Request {
		return &Request{
			uniqueQuery: make([]byte, 0, preallocatedBufferCapacity+(maxTagsLen+tagBufferCapacity)),
		}
	})
	KeyBufferPool = synced.NewBatchPool[*types.SizedBox[[]byte]](synced.PreallocateBatchSize, func() *types.SizedBox[[]byte] {
		return &types.SizedBox[[]byte]{
			Value: make([]byte, 0, preallocatedBufferCapacity),
			CalcWeightFn: func(box *types.SizedBox[[]byte]) int64 {
				return int64(cap(box.Value)) + consts.PtrBytesWeight
			},
		}
	})
	TagsSlicesPool = synced.NewBatchPool[*types.SizedBox[[][]byte]](synced.PreallocateBatchSize, func() *types.SizedBox[[][]byte] {
		batch := make([][]byte, maxTagsLen)
		for i := range batch {
			batch[i] = make([]byte, 0, tagBufferCapacity)
		}
		return &types.SizedBox[[][]byte]{
			Value: batch,
			CalcWeightFn: func(box *types.SizedBox[[][]byte]) int64 {
				weight := int64(unsafe.Sizeof(box.Value))
				for _, tag := range box.Value {
					weight += int64(cap(tag)) + int64(unsafe.Sizeof(tag))
				}
				return int64(cap(box.Value))
			},
		}
	})
)

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

func (r *Request) Weight() int64 {
	weight := int(unsafe.Sizeof(*r))
	weight += len(r.project)
	weight += len(r.domain)
	weight += len(r.language)
	weight += len(r.uniqueQuery)
	weight += len(r.tags)
	for _, tag := range r.tags {
		weight += len(tag)
	}
	return int64(weight)
}

// NewManualRequest creates a Request with explicit parameters, bypassing fasthttp.
func NewManualRequest(project, domain, language []byte, tags [][]byte) (*Request, error) {
	r, err := RequestsPool.Get().clear().setUp(project, domain, language, tags, func() {}).validate()
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
	r, err := RequestsPool.Get().clear().setUp(project, domain, language, tags, releaseFn).validate()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// setUp initializes the Request, interns all fields, builds keys, and sets up uniqueQuery.
func (r *Request) setUp(project, domain, language []byte, tags [][]byte, releaseFn func()) *Request {
	// Strict interning: all input slices are deduped
	r.project = project
	r.domain = domain
	r.language = language

	// Intern each tag value
	for i, tag := range tags {
		if len(tag) > 0 {
			tags[i] = tag
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

	tagsSls := TagsSlicesPool.Get()
	// Reset each tag slice
	for i := range tagsSls.Value {
		tagsSls.Value[i] = tagsSls.Value[i][:0]
	}

	i := 0
	args.VisitAll(func(key, value []byte) {
		if !bytes.HasPrefix(key, choiceValue) || bytes.Equal(value, nullValue) {
			return
		}
		tagsSls.Value[i] = value
		i++
	})

	// Only return used tags
	tags := tagsSls.Value[:i]
	return tags, func() { TagsSlicesPool.Put(tagsSls) }
}

// Getters for all important fields.
func (r *Request) GetProject() []byte  { return r.project }
func (r *Request) GetDomain() []byte   { return r.domain }
func (r *Request) GetLanguage() []byte { return r.language }
func (r *Request) GetTags() [][]byte   { return r.tags }

// setUpKey computes a unique hash key (xxh3) for this request.
func (r *Request) setUpKey() uint64 {
	buf := KeyBufferPool.Get()
	defer KeyBufferPool.Put(buf)
	buf.Value = buf.Value[:0]

	buf.Value = append(buf.Value, r.project...)
	buf.Value = append(buf.Value, r.domain...)
	buf.Value = append(buf.Value, r.language...)
	for _, tag := range r.tags {
		if len(tag) == 0 {
			continue
		}
		buf.Value = append(buf.Value, tag...)
	}

	hasher := HasherPool.Get()
	defer HasherPool.Put(hasher)
	hasher.Value.Reset()
	if _, err := hasher.Value.Write(buf.Value); err != nil {
		panic(err)
	}

	key := hasher.Value.Sum64()
	r.key = key
	return key
}

// Key returns the computed hash key for the request.
func (r *Request) Key() uint64 { return atomic.LoadUint64(&r.key) }

// ShardKey returns the precomputed shard index.
func (r *Request) ShardKey() uint64 { return atomic.LoadUint64(&r.shardKey) }

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
	r.releaseFn()
	r.clear()
	RequestsPool.Put(r)
}
