package model

import (
	"github.com/zeebo/xxh3"
	"strings"
	"sync"
)

type Request struct {
	mu       *sync.RWMutex
	project  string
	domain   string
	language string
	choice   string
	// calculated fields
	uniqueKey    uint64
	uniqueString string
}

func NewRequest(project string, domain, language, choice string) *Request {
	return &Request{
		mu:       &sync.RWMutex{},
		project:  project,
		domain:   domain,
		language: language,
		choice:   choice,
	}
}

func (r *Request) GetProject() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.project
}
func (r *Request) GetDomain() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.domain
}
func (r *Request) GetLanguage() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.language
}
func (r *Request) GetChoice() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.choice
}
func (r *Request) String() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.uniqueString != "" {
		return r.uniqueString
	}

	l := 3
	l += len(r.project)
	l += len(r.domain)
	l += len(r.language)
	l += len(r.choice)

	var b strings.Builder
	b.Grow(l)

	b.WriteString(r.project)
	b.WriteString(",")
	b.WriteString(r.domain)
	b.WriteString(",")
	b.WriteString(r.language)
	b.WriteString(",")
	b.WriteString(r.choice)
	r.uniqueString = b.String()

	return r.uniqueString
}
func (r *Request) UniqueKey() uint64 {
	if r.uniqueKey == 0 {
		r.uniqueKey = xxh3.HashString(r.String())
	}
	return r.uniqueKey
}
