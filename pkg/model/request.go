package model

import (
	"bytes"
	"errors"
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
	r, err := new(Request).setUp(project, domain, language, tags).validate()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// NewRequest builds a Request from fasthttp.Args, with strict interning and pooling.
func NewRequest(q *fasthttp.Args) (*Request, error) {
	var (
		project  = q.Peek("project[id]")
		domain   = q.Peek("domain")
		language = q.Peek("language")
		tags     = extractTags(q)
	)
	r, err := new(Request).setUp(project, domain, language, tags).validate()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// setUp initializes the Request, interns all fields, builds keys, and sets up uniqueQuery.
func (r *Request) setUp(project, domain, language []byte, tags [][]byte) *Request {
	r.project = project
	r.domain = domain
	r.language = language
	r.tags = tags

	r.setUpQuery()
	r.setUpShardKey(r.setUpKey())

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
func extractTags(args *fasthttp.Args) [][]byte {
	var (
		nullValue   = []byte("null")
		choiceValue = []byte("choice")
	)

	i := 0
	tags := make([][]byte, 0, 10)
	args.VisitAll(func(key, tag []byte) {
		if !bytes.HasPrefix(key, choiceValue) || bytes.Equal(tag, nullValue) {
			return
		}
		tags = append(tags, tag)
		i++
	})

	return tags
}

// Getters for all important fields.
func (r *Request) GetProject() []byte  { return r.project }
func (r *Request) GetDomain() []byte   { return r.domain }
func (r *Request) GetLanguage() []byte { return r.language }
func (r *Request) GetTags() [][]byte   { return r.tags }

// setUpKey computes a unique hash key (xxh3) for this request.
func (r *Request) setUpKey() uint64 {
	buf := make([]byte, 0, preallocatedBufferCapacity+(tagBufferCapacity*maxTagsLen))
	buf = append(buf, r.project...)
	buf = append(buf, r.domain...)
	buf = append(buf, r.language...)
	for _, tag := range r.tags {
		if len(tag) == 0 {
			continue
		}
		buf = append(buf, tag...)
	}

	hasher := HasherPool.Get()
	defer HasherPool.Put(hasher)
	hasher.Value.Reset()
	if _, err := hasher.Value.Write(buf); err != nil {
		panic(err)
	}

	r.key = hasher.Value.Sum64()
	return r.key
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
	buf := make([]byte, 0, preallocatedBufferCapacity+(tagBufferCapacity*maxTagsLen))
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
