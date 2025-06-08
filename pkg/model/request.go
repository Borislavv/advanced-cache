// Переписанная версия model/request.go и model/response.go с упором на безопасный доступ, sync.Pool, без лишних аллокаций.

package model

import (
	"bytes"
	"errors"
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/xxh3"
	"sync"
	"sync/atomic"
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
	slBtsPool = synced.NewBatchPool[*bytes.Buffer](synced.PreallocationBatchSize, func() *bytes.Buffer {
		return new(bytes.Buffer)
	})
	QueryPool = synced.NewBatchPool[*Query](synced.PreallocationBatchSize, func() *Query {
		return &Query{}
	})
	RequestsPool = synced.NewBatchPool[*Request](synced.PreallocationBatchSize, func() *Request {
		return &Request{
			query:       &atomic.Pointer[Query]{},
			uniqueQuery: &atomic.Pointer[bytes.Buffer]{},
			uniqueKey:   0,
		}
	})
	TagsSlicesPool = synced.NewBatchPool[[][]byte](synced.PreallocationBatchSize, func() [][]byte {
		batch := make([][]byte, maxTagsLen)
		for i := range batch {
			batch[i] = make([]byte, tagBufferCapacity)
		}
		return batch
	})
)

type Query struct {
	project, domain, language []byte
	tags                      [][]byte
	putTagsInPoolFn           func()
}

func (q *Query) Values() (project []byte, domain []byte, language []byte, tags [][]byte) {
	return q.project, q.domain, q.language, q.tags
}

func (q *Query) free() {
	if q.putTagsInPoolFn != nil {
		q.putTagsInPoolFn()
	}
	QueryPool.Put(q)
}

type Request struct {
	query       *atomic.Pointer[Query]
	uniqueQuery *atomic.Pointer[bytes.Buffer]
	uniqueKey   uint64
}

func NewRequest(args *fasthttp.Args) (*Request, error) {
	var (
		project               = internBytes(args.Peek("project[id]"))
		domain                = internBytes(args.Peek("domain"))
		language              = internBytes(args.Peek("language"))
		tags, putTagsInPoolFn = extractTags(args)
	)
	if len(project) == 0 || len(domain) == 0 || len(language) == 0 {
		return nil, errors.New("missing required arguments")
	}
	req := RequestsPool.Get()
	query := QueryPool.Get()
	*query = Query{
		project:         project,
		domain:          domain,
		language:        language,
		tags:            tags,
		putTagsInPoolFn: putTagsInPoolFn,
	}
	atomic.StoreUint64(&req.uniqueKey, 0)
	req.uniqueQuery.Store(nil)
	req.query.Store(query)
	return req, nil
}

func (r *Request) ToQuery() []byte {
	if q := r.uniqueQuery.Load(); q != nil {
		if b := q.Bytes(); len(b) > 0 {
			return b
		}
	}

	var project, domain, language, tags = r.query.Load().Values()

	buf := slBtsPool.Get()
	buf.Reset()

	buf.Write([]byte("?project[id]="))
	buf.Write(project)
	buf.Write([]byte("&domain="))
	buf.Write(domain)
	buf.Write([]byte("&language="))
	buf.Write(language)

	var n int
	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}
		buf.Write([]byte("&choice"))
		buf.Write(bytes.Repeat([]byte("[choice]"), n))
		buf.Write([]byte("[name]="))
		buf.Write(tag)
		n++
	}
	buf.Write([]byte("&choice"))
	buf.Write(bytes.Repeat([]byte("[choice]"), n))
	buf.Write([]byte("=null"))

	r.uniqueQuery.Store(buf)

	return buf.Bytes()
}

func (r *Request) UniqueKey() uint64 {
	if key := atomic.LoadUint64(&r.uniqueKey); key != 0 {
		return key
	}

	var project, domain, language, tags = r.query.Load().Values()

	buf := slBtsPool.Get()
	defer slBtsPool.Put(buf)
	buf.Reset()

	buf.Write(project)
	buf.Write(domain)
	buf.Write(language)
	for _, tag := range tags {
		buf.Write(tag)
	}

	h := hasherPool.Get()
	defer hasherPool.Put(h)

	h.Reset()
	_, e := h.Write(buf.Bytes())
	if e != nil {
		log.Err(e).Str("buf", string(buf.Bytes())).Msg("failed to compute hash")
	}
	key := h.Sum64()

	atomic.StoreUint64(&r.uniqueKey, key)

	return key
}

func extractTags(args *fasthttp.Args) ([][]byte, func()) {
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
		tagsSls[idx] = append(tagsSls[idx], internBytes(value)...)
		idx++
	})
	return tagsSls[:idx], func() {
		TagsSlicesPool.Put(tagsSls)
	}
}

var internPool sync.Map

func internBytes(b []byte) []byte {
	strKey := string(b)
	if val, ok := internPool.Load(strKey); ok {
		return val.([]byte)
	}
	actual, _ := internPool.LoadOrStore(strKey, b)
	return actual.([]byte)
}

func (r *Request) Free() {
	query := r.query.Load()
	if query != nil {
		query.free()
	}

	buf := r.uniqueQuery.Load()
	if buf != nil {
		slBtsPool.Put(buf)
	}

	RequestsPool.Put(r)
}
