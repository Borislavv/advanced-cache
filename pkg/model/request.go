package model

import (
	"bytes"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/xxh3"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
)

const (
	// preallocated capacities for intermediate byte slices
	preallocQueryCapacity = 256
	preallocKeyCapacity   = 256

	// maximum number of tags to extract
	maxTagsLen = 10
)

var (
	// Pool of hashers for computing UniqueKey
	hasherPool = synced.NewBatchPool[*xxh3.Hasher](
		synced.PreallocationBatchSize,
		func() *xxh3.Hasher { return xxh3.New() },
	)

	// Pool of Query objects to reduce allocations
	QueryPool = synced.NewBatchPool[*Query](
		synced.PreallocationBatchSize,
		func() *Query { return &Query{} },
	)

	// Pool of Request objects
	RequestsPool = synced.NewBatchPool[*Request](
		synced.PreallocationBatchSize,
		func() *Request {
			return &Request{
				query:       &atomic.Pointer[Query]{},
				uniqueQuery: &atomic.Pointer[[]byte]{},
			}
		},
	)

	// Pool of reusable tag-slice batches
	TagsSlicesPool = synced.NewBatchPool[[][]byte](
		synced.PreallocationBatchSize,
		func() [][]byte {
			batch := make([][]byte, maxTagsLen)
			for i := range batch {
				batch[i] = make([]byte, 0)
			}
			return batch
		},
	)
)

// Query holds the parsed arguments for one HTTP request,
// along with a cleanup function to return the tag buffers to their pool.
type Query struct {
	project, domain, language []byte
	tags                      [][]byte
	putTagsInPoolFn           func()
}

// Values returns the parsed components of the query.
func (q *Query) Values() (project, domain, language []byte, tags [][]byte) {
	return q.project, q.domain, q.language, q.tags
}

// free returns any resources held by Query back to their pools.
func (q *Query) free() {
	if q.putTagsInPoolFn != nil {
		q.putTagsInPoolFn()
	}
	QueryPool.Put(q)
}

// Request represents a single HTTP request in the cache layer.
// It caches the generated query-string ([]byte) and UniqueKey (uint64)
// to avoid repeated allocations on hot paths.
type Request struct {
	// atomic pointer to the parsed Query
	query *atomic.Pointer[Query]

	// atomic pointer to the cached output of ToQuery()
	// stores a *[]byte, nil if not yet computed
	uniqueQuery *atomic.Pointer[[]byte]

	// cached hash key for quick lookups
	uniqueKey uint64
}

// NewRequest parses the fasthttp.Args and returns a pooled Request.
// Required arguments are “project[id]”, “domain” and “language”.
// Any missing parameter yields an error.
func NewRequest(args *fasthttp.Args) (*Request, error) {
	project := internBytes(args.Peek("project[id]"))
	domain := internBytes(args.Peek("domain"))
	language := internBytes(args.Peek("language"))

	tags, putTagsFn := extractTags(args)

	if len(project) == 0 || len(domain) == 0 || len(language) == 0 {
		return nil, errors.New("missing required arguments")
	}

	// Acquire Request object from pool
	req := RequestsPool.Get()

	// Parse and store Query
	q := QueryPool.Get()
	*q = Query{
		project:         project,
		domain:          domain,
		language:        language,
		tags:            tags,
		putTagsInPoolFn: putTagsFn,
	}

	req.query.Store(q)
	req.uniqueQuery.Store(nil)
	atomic.StoreUint64(&req.uniqueKey, 0)

	return req, nil
}

// ToQuery builds the query-string once and caches the []byte result.
// Subsequent calls return the same slice without reallocations.
func (r *Request) ToQuery() []byte {
	if ptr := r.uniqueQuery.Load(); ptr != nil {
		return *ptr
	}

	project, domain, language, tags := r.query.Load().Values()

	// Build into a local byte slice with preallocated capacity
	b := make([]byte, 0, preallocQueryCapacity)
	b = append(b, "?project[id]="...)
	b = append(b, project...)
	b = append(b, "&domain="...)
	b = append(b, domain...)
	b = append(b, "&language="...)
	b = append(b, language...)

	// Append tags in user-defined order
	var n int
	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}
		b = append(b, "&choice"...)
		for i := 0; i < n; i++ {
			b = append(b, "[choice]"...)
		}
		b = append(b, "[name]="...)
		b = append(b, tag...)
		n++
	}

	// Terminate the choices list
	b = append(b, "&choice"...)
	for i := 0; i < n; i++ {
		b = append(b, "[choice]"...)
	}
	b = append(b, "=null"...)

	// Cache and return
	r.uniqueQuery.Store(&b)
	return b
}

// UniqueKey computes a stable uint64 hash for the request components.
// It caches the result after the first computation.
func (r *Request) UniqueKey() uint64 {
	if k := atomic.LoadUint64(&r.uniqueKey); k != 0 {
		return k
	}

	project, domain, language, tags := r.query.Load().Values()

	// Build a local buffer for hashing
	buf := make([]byte, 0, preallocKeyCapacity)
	buf = append(buf, project...)
	buf = append(buf, domain...)
	buf = append(buf, language...)
	for _, tag := range tags {
		buf = append(buf, tag...)
	}

	// Hash using pooled xxh3.Hasher
	h := hasherPool.Get()
	defer hasherPool.Put(h)

	h.Reset()
	if _, err := h.Write(buf); err != nil {
		log.Err(err).Str("buf", string(buf)).Msg("failed to compute hash")
	}

	key := h.Sum64()
	atomic.StoreUint64(&r.uniqueKey, key)
	return key
}

// Free returns the Request and its Query back to their pools.
// It also clears the cached query-string slice.
func (r *Request) Free() {
	if q := r.query.Load(); q != nil {
		q.free()
	}

	// Clear the cached query bytes
	r.uniqueQuery.Store(nil)

	//RequestsPool.Put(r)
}

// extractTags reads all “choice” parameters from args,
// returns the [][]byte of tag values and a cleanup callback.
func extractTags(args *fasthttp.Args) ([][]byte, func()) {
	nullValue := []byte("null")
	choiceKey := []byte("choice")

	// Get a reusable batch of slices
	tagsSls := TagsSlicesPool.Get()
	for i := range tagsSls {
		tagsSls[i] = tagsSls[i][:0]
	}

	// Collect up to maxTagsLen valid tags
	idx := 0
	args.VisitAll(func(key, value []byte) {
		if !bytes.HasPrefix(key, choiceKey) || bytes.Equal(value, nullValue) || idx >= maxTagsLen {
			return
		}
		tagsSls[idx] = append(tagsSls[idx], internBytes(value)...)
		idx++
	})

	// Slice to actual length
	tags := tagsSls[:idx]
	return tags, func() {
		TagsSlicesPool.Put(tagsSls)
	}
}

var internPool sync.Map

// internBytes de-dupes equal byte-slices via a sync.Map.
// This reduces overall allocations at the cost of some map lookups.
func internBytes(b []byte) []byte {
	key := string(b)
	if v, ok := internPool.Load(key); ok {
		return v.([]byte)
	}
	actual, _ := internPool.LoadOrStore(key, b)
	return actual.([]byte)
}
