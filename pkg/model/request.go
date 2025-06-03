package model

import (
	"bytes"
	"errors"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/xxh3"
	"go.uber.org/atomic"
	"sync"
)

const preallocatedQueryBufCapacity = 170 // for pre-allocated buffer for concat string literals and url

type Request struct {
	mu       *sync.RWMutex
	project  []byte
	domain   []byte
	language []byte
	tags     [][]byte
	// calculated fields (calling while init.)
	uniqueQuery []byte
	uniqueKey   *atomic.Uint64
}

func NewRequest(q *fasthttp.Args) (*Request, error) {
	project := q.Peek("project[id]")
	if project == nil || len(project) == 0 {
		return nil, errors.New("project is not specified")
	}
	domain := q.Peek("domain")
	if domain == nil || len(domain) == 0 {
		return nil, errors.New("domain is not specified")
	}
	language := q.Peek("language")
	if language == nil || len(language) == 0 {
		return nil, errors.New("language is not specified")
	}
	return (&Request{
		mu:        &sync.RWMutex{},
		project:   project,
		domain:    domain,
		language:  language,
		tags:      ExtractTags(q),
		uniqueKey: &atomic.Uint64{},
	}).warmUp(), nil
}

func NewManualRequest(project, domain, language []byte, tags [][]byte) (*Request, error) {
	if project == nil || len(project) == 0 {
		return nil, errors.New("project is not specified")
	}
	if domain == nil || len(domain) == 0 {
		return nil, errors.New("domain is not specified")
	}
	if language == nil || len(language) == 0 {
		return nil, errors.New("language is not specified")
	}
	return (&Request{
		mu:        &sync.RWMutex{},
		project:   project,
		domain:    domain,
		language:  language,
		tags:      tags,
		uniqueKey: &atomic.Uint64{},
	}).warmUp(), nil
}

// ExtractTags - returns a slice with []byte("${choice name}").
func ExtractTags(args *fasthttp.Args) [][]byte {
	var (
		nullValue   = []byte("null")
		choiceValue = []byte("choice")
	)

	var tags [][]byte
	args.VisitAll(func(key, value []byte) {
		if !bytes.HasPrefix(key, choiceValue) || bytes.Equal(value, nullValue) {
			return
		}
		tags = append(tags, value)
	})
	return tags
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
	key := r.uniqueKey.Load()
	if key != 0 {
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
		buf = append(buf, tag...)
	}
	// calculate hash of unique []byte
	key = xxh3.Hash(buf)
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

	bufCap := preallocatedQueryBufCapacity + len(r.project) + len(r.domain) + len(r.language)
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
	var choiceArrBytes = []byte("[choice]")
	for _, tag := range r.tags {
		buf = append(buf, []byte("&choice")...)
		buf = append(buf, bytes.Repeat(choiceArrBytes, n)...)
		buf = append(buf, []byte("[name]=")...)
		buf = append(buf, tag...)
		n++
	}
	buf = append(buf, []byte("&choice")...)
	buf = append(buf, bytes.Repeat(choiceArrBytes, n)...)
	buf = append(buf, []byte("=null")...)

	r.mu.Lock()
	r.uniqueQuery = buf
	r.mu.Unlock()

	return buf
}

func (r *Request) warmUp() *Request {
	_ = r.ToQuery()
	_ = r.UniqueKey()
	return r
}
