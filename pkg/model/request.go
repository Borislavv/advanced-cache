package model

import (
	"hash/maphash"
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
	uniqueQuery  string
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
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.strUnlocked()
}
func hashKey(key string, hasher *maphash.Hash) uint64 {
	hasher.Reset()
	_, _ = hasher.WriteString(key)
	return hasher.Sum64()
}
func (r *Request) UniqueKey(hasher *maphash.Hash) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.uniqueKey == 0 {
		r.uniqueKey = hashKey(r.strUnlocked(), hasher)
	}
	return r.uniqueKey
}
func (r *Request) strUnlocked() string {
	if r.uniqueString == "" {
		r.uniqueString = r.project + "," + r.domain + "," + r.language + "," + r.choice
	}
	return r.uniqueString
}
func (r *Request) ToQuery() (string, error) {
	r.mu.RLock()
	project := r.project
	domain := r.domain
	language := r.language
	tags, err := ExtractTags(r.choice)
	if err != nil {
		r.mu.RUnlock()
		return "", err
	}
	r.mu.RUnlock()

	if r.uniqueQuery == "" {
		r.uniqueQuery = "?project[id]=" + project + "&domain=" + domain + "&language=" + language

		l := len(tags) - 1
		for i := l; i >= 0; i-- {
			name := tags[i]
			r.uniqueQuery += "&choice[name]=" + name
			if i-1 >= 0 {
				r.uniqueQuery += "&choice[choice]=" + tags[i-1]
			} else {
				r.uniqueQuery += "&choice[choice]=null"
			}
		}
	}

	return r.uniqueQuery, nil
}
