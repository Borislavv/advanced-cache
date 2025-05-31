package model

import (
	"fmt"
	"github.com/zeebo/xxh3"
	"sync"
)

type Request struct {
	mu       *sync.RWMutex
	Project  string
	domain   string
	language string
	choice   string
}

func NewRequest(project string, domain, language, choice string) *Request {
	return &Request{
		mu:       &sync.RWMutex{},
		Project:  project,
		domain:   domain,
		language: language,
		choice:   choice,
	}
}

func (r *Request) GetProject() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Project
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
	return fmt.Sprintf("%s,%s,%s,%s", r.Project, r.domain, r.language, r.choice)
}
func (r *Request) UniqueKey() uint64 {
	return xxh3.HashString(r.String())
}
