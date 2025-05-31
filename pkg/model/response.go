package model

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const nameToken = "name"

type Response struct {
	mu                 *sync.RWMutex
	item               *list.Element
	request            *Request
	data               []byte
	tags               []string
	frequency          int // number of times of response was accessed
	createdAt          time.Time
	lastAccess         time.Time
	revalidatedAt      time.Time
	revalidateInterval time.Duration
}

func NewResponse(item *list.Element, req *Request, data []byte, revalidateInterval time.Duration) (*Response, error) {
	tags, err := ExtractTags(req.GetChoice())
	if err != nil {
		return nil, fmt.Errorf("cannot extract tags from choice: %s", err.Error())
	}
	return &Response{
		mu:                 &sync.RWMutex{},
		request:            req,
		item:               item,
		data:               data,
		tags:               tags,
		frequency:          0,
		createdAt:          time.Now(),
		lastAccess:         time.Now(),
		revalidatedAt:      time.Now(),
		revalidateInterval: revalidateInterval,
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
	return r.item
}
func (r *Response) SetListElement(el *list.Element) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.item = el
}
func (r *Response) Copy(source *Response) *Response {
	request := source.GetRequest()
	frequency := source.GetFrequency()
	lastAccess := source.GetLastAccess()
	data := source.GetData()
	tags := source.GetTags()
	revalidatedAt := source.GetRevalidatedAt()

	r.mu.Lock()
	r.request = request
	r.frequency = frequency
	r.lastAccess = lastAccess
	r.data = data
	r.tags = tags
	r.revalidatedAt = revalidatedAt
	r.mu.Unlock()

	return r
}
func (r *Response) Size() uintptr {
	return unsafe.Sizeof(*r)
}
func ExtractTags(choice string) ([]string, error) {
	var names []string
	var expectValue bool

	decoder := json.NewDecoder(strings.NewReader(choice))

	for {
		t, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch token := t.(type) {
		case string:
			if expectValue {
				names = append(names, token)
				expectValue = false
			} else if token == nameToken {
				expectValue = true
			}
		default:
			expectValue = false
		}
	}

	return names, nil
}
