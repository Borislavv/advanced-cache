package list

import (
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"sync"
)

// Element is a node in the doubly linked list that holds a value of type T.
type Element[T any] struct {
	next, prev *Element[T]
	list       *List[T]
	Value      T
}

// Next returns the next element in the list or nil if it's the end or unlinked.
func (e *Element[T]) Next() *Element[T] {
	if p := e.next; e.list != nil && p != e.list.root {
		return p
	}
	return nil
}

// Prev returns the previous element in the list or nil if it's the beginning or unlinked.
func (e *Element[T]) Prev() *Element[T] {
	if p := e.prev; e.list != nil && p != e.list.root {
		return p
	}
	return nil
}

// List is a generic doubly linked list implementation with optional thread-safety and element pooling.
type List[T any] struct {
	isGuarded bool
	mu        sync.RWMutex
	root      *Element[T]
	elemPool  *synced.BatchPool[*Element[T]]
}

// New creates a new List instance. If isMustBeListAThreadSafe is true, it uses a mutex for concurrency safety.
func New[T any](isMustBeListAThreadSafe bool) *List[T] {
	l := &List[T]{
		isGuarded: isMustBeListAThreadSafe,
		elemPool: synced.NewBatchPool[*Element[T]](synced.PreallocationBatchSize, func() *Element[T] {
			return &Element[T]{}
		}),
	}
	l.Init()
	return l
}

// Init initializes the internal root sentinel node of the list.
func (l *List[T]) Init() *List[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}

	l.root.next = l.root
	l.root.prev = l.root
	return l
}

// Front returns the first element in the list.
func (l *List[T]) Front() *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if l.root != nil {
		if next := l.root.next; next != nil {
			return next
		}
	}
	return nil
}

// Back returns the last element in the list.
func (l *List[T]) Back() *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if l.root != nil {
		if prev := l.root.prev; prev != nil {
			return prev
		}
	}
	return nil
}

// insert adds an element e after the element at.
func (l *List[T]) insert(e, at *Element[T]) *Element[T] {
	e.prev = at
	e.next = at.next
	at.next.prev = e
	at.next = e
	e.list = l
	return e
}

// remove disconnects the element from the list and returns it to the pool.
func (l *List[T]) remove(e *Element[T]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.elemPool.Put(e)
}

// move relocates element e after element at within the list.
func (l *List[T]) move(e, at *Element[T]) {
	if e == at {
		return
	}

	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	at.next.prev = e
	at.next = e
}

// Remove deletes the element from the list and returns its value.
func (l *List[T]) Remove(e *Element[T]) (v T) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if e.list == l {
		v = e.Value
		l.remove(e)
	}
	return
}

// PushFront inserts a value at the front of the list.
func (l *List[T]) PushFront(v T) *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	elem := l.elemPool.Get()
	*elem = Element[T]{Value: v}
	return l.insert(elem, l.root)
}

// PushBack inserts a value at the end of the list.
func (l *List[T]) PushBack(v T) *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	elem := l.elemPool.Get()
	*elem = Element[T]{Value: v}
	return l.insert(elem, l.root.prev)
}

// MoveToFront moves the specified element to the front of the list.
func (l *List[T]) MoveToFront(e *Element[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if e.list != l || l.root.next == e {
		return
	}
	l.move(e, l.root)
}

// MoveToBack moves the specified element to the end of the list.
func (l *List[T]) MoveToBack(e *Element[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if e.list != l || l.root.prev == e {
		return
	}
	l.move(e, l.root.prev)
}

// SwapValues swaps the values between two elements.
func (l *List[T]) SwapValues(e1, e2 *Element[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	e1.Value, e2.Value = e2.Value, e1.Value
}

// InsertAfter inserts a value after the given element in the list.
func (l *List[T]) InsertAfter(v T, mark *Element[T]) *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if mark.list != l {
		return nil
	}
	elem := l.elemPool.Get()
	*elem = Element[T]{Value: v}
	return l.insert(elem, mark)
}

// InsertBefore inserts a value before the given element in the list.
func (l *List[T]) InsertBefore(v T, mark *Element[T]) *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if mark.list != l {
		return nil
	}
	elem := l.elemPool.Get()
	*elem = Element[T]{Value: v}
	return l.insert(elem, mark.prev)
}

// MoveBefore moves element e before the mark element in the list.
func (l *List[T]) MoveBefore(e, mark *Element[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.prev)
}

// MoveAfter moves element e after the mark element in the list.
func (l *List[T]) MoveAfter(e, mark *Element[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark)
}

// PushBackList appends a copy of another list to the end of this list.
func (l *List[T]) PushBackList(other *List[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	for e := other.Front(); e != nil; e = e.Next() {
		elem := l.elemPool.Get()
		*elem = Element[T]{Value: e.Value}
		l.insert(elem, l.root.prev)
	}
}

// PushFrontList prepends a copy of another list to the front of this list.
func (l *List[T]) PushFrontList(other *List[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	for e := other.Back(); e != nil; e = e.Prev() {
		elem := l.elemPool.Get()
		*elem = Element[T]{Value: e.Value}
		l.insert(elem, l.root)
	}
}
