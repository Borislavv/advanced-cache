package model

import (
	"github.com/zeebo/xxh3"
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
	r.uniqueString = r.project + "," + r.domain + "," + r.language + "," + r.choice
	return r.uniqueString
}
func (r *Request) UniqueKey() uint64 {
	if r.uniqueKey == 0 {
		r.uniqueKey = xxh3.HashString(r.String())
	}
	return r.uniqueKey
}
