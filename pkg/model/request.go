package model

import (
	"github.com/zeebo/xxh3"
	"go.uber.org/atomic"
	"sync"
)

const preallocatedQueryBufCapacity = 100 // for pre-allocated buffer for concat string literals and url

var (
	nullValue      = []byte("null")
	choiceValue    = []byte("choice")
	arrChoiceValue = []byte("[choice]")
)

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

func NewRequest(project, domain, language []byte, tags [][]byte) *Request {
	return (&Request{
		mu:        &sync.RWMutex{},
		project:   project,
		domain:    domain,
		language:  language,
		tags:      tags,
		uniqueKey: &atomic.Uint64{},
	}).warmUp()
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
	for i := 0; i < len(r.tags); i++ {
		bufCap += len(r.tags[i])
	}

	buf := make([]byte, 0, bufCap)
	buf = append(buf, []byte("?project[id]=")...)
	buf = append(buf, r.project...)
	buf = append(buf, []byte("&domain=")...)
	buf = append(buf, r.domain...)
	buf = append(buf, []byte("&language=")...)
	buf = append(buf, r.language...)

	l := len(r.tags) - 1
	for i := l; i >= 0; i-- {
		name := r.tags[i]
		buf = append(buf, []byte("&choice[name]=")...)
		buf = append(buf, name...)
		if i+1 < l {
			buf = append(buf, []byte("&choice[choice]=")...)
			buf = append(buf, r.tags[i+1]...)
		}
	}

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
