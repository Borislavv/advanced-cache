package model

import (
	"container/list"
	"fmt"
	"github.com/buger/jsonparser"
	"sync"
	"time"
	"unsafe"
)

const nameToken = "name"

type Response struct {
	Meta
	mu          *sync.RWMutex
	listElement *list.Element
	lastAccess  time.Time
	createdAt   time.Time
}

type Meta struct {
	request            *Request
	frequency          int // number of times of response was accessed
	data               []byte
	tags               []string
	revalidatedAt      time.Time
	revalidateInterval time.Duration
}

func NewResponse(item *list.Element, req *Request, data []byte, revalidateInterval time.Duration) (*Response, error) {
	tags, err := ExtractTags(req.GetChoice())
	if err != nil {
		return nil, fmt.Errorf("cannot extract tags from choice: %s", err.Error())
	}
	return &Response{
		mu:          &sync.RWMutex{},
		listElement: item,
		createdAt:   time.Now(),
		lastAccess:  time.Now(),
		Meta: Meta{
			request:            req,
			data:               data,
			tags:               tags,
			revalidatedAt:      time.Now(),
			revalidateInterval: revalidateInterval,
		},
	}, nil
}
func (r *Response) GetRequest() *Request {
	r.mu.RLock()
	req := r.request
	r.mu.RUnlock()
	return req
}
func (r *Response) SetRequest(req *Request) {
	r.mu.Lock()
	r.request = req
	r.mu.Unlock()
}
func (r *Response) GetData() []byte {
	r.mu.RLock()
	data := r.data
	r.mu.RUnlock()
	return data
}
func (r *Response) SetData(data []byte) {
	r.mu.Lock()
	r.data = data
	r.revalidatedAt = time.Now()
	r.mu.Unlock()
}
func (r *Response) GetTags() []string {
	r.mu.RLock()
	tags := r.tags
	r.mu.RUnlock()
	return tags
}
func (r *Response) SetTags(tags []string) {
	r.mu.Lock()
	r.tags = tags
	r.mu.Unlock()
}
func (r *Response) GetFrequency() int {
	r.mu.RLock()
	frequency := r.frequency
	r.mu.RUnlock()
	return frequency
}
func (r *Response) SetFrequency(frequency int) {
	r.mu.Lock()
	r.frequency = frequency
	r.mu.Unlock()
}
func (r *Response) GetLastAccess() time.Time {
	r.mu.RLock()
	lastAccess := r.lastAccess
	r.mu.RUnlock()
	return lastAccess
}
func (r *Response) SetLastAccess() {
	r.mu.Lock()
	r.lastAccess = time.Now()
	r.mu.Unlock()
}
func (r *Response) GetCreatedAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.createdAt
}
func (r *Response) GetRevalidatedAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.revalidatedAt
}
func (r *Response) SetRevalidatedAt() {
	r.mu.Lock()
	r.revalidatedAt = time.Now()
	r.mu.Unlock()
}
func (r *Response) GetRevalidateInterval() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.revalidateInterval
}
func (r *Response) Touch() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastAccess = time.Now()
	r.frequency = r.frequency + 1
}
func (r *Response) GetListElement() *list.Element {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.listElement
}
func (r *Response) SetListElement(el *list.Element) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listElement = el
}

func (r *Response) GetMeta() Meta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Meta
}

func (r *Response) SetMeta(meta Meta) {
	r.mu.Lock()
	r.Meta = meta
	r.mu.Unlock()
}

func (r *Response) Size() uintptr {
	return unsafe.Sizeof(*r)
}
func ExtractTags(choice string) ([]string, error) {
	names := make([]string, 0, 7)

	if err := jsonparser.ObjectEach([]byte(choice), func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if string(key) == nameToken {
			names = append(names, string(value))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return names, nil
}
