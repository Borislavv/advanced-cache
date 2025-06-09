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
	len       int
	isGuarded bool
	mu        sync.Mutex
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
	l.init()
	return l
}

// init initializes or clears list l.
func (l *List[T]) init() *List[T] {
	root := l.elemPool.Get()
	*root = Element[T]{}
	l.root = root
	l.root.next = l.root
	l.root.prev = l.root
	l.len = 0
	return l
}

// Len returns the number of elements of list l.
// The complexity is O(1).
func (l *List[T]) Len() int {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	return l.len
}

func (l *List[T]) SwapValues(a, b *Element[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	a.Value, b.Value = b.Value, a.Value
}

// Front returns the first element of list l or nil if the list is empty.
func (l *List[T]) Front() *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

// Back returns the last element of list l or nil if the list is empty.
func (l *List[T]) Back() *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// insert inserts e after at, increments l.len, and returns e.
func (l *List[T]) insert(e, at *Element[T]) *Element[T] {
	if at != nil {
		e.prev = at
		e.next = at.next
	}

	if e.prev != nil {
		e.prev.next = e
		e.next.prev = e
	}

	e.list = l
	l.len++

	return e
}

// insertValue is a convenience wrapper for insert(&Element{Value: v}, at).
func (l *List[T]) insertValue(v T, at *Element[T]) *Element[T] {
	el := l.elemPool.Get()
	*el = Element[T]{Value: v}
	return l.insert(el, at)
}

// remove removes e from its list, decrements l.len
func (l *List[T]) remove(e *Element[T]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	l.elemPool.Put(e)
	l.len--
}

// move moves e to next to at.
func (l *List[T]) move(e, at *Element[T]) {
	if e == at {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
}

// Remove removes e from l if e is an element of list l.
// It returns the element value e.Value.
// The element must not be nil.
func (l *List[T]) Remove(e *Element[T]) any {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if e == nil {
		return nil
	}
	if e.list == l {
		// if e.list == l, l must have been initialized when e was inserted
		// in l or l == nil (e is a zero Element) and l.remove will crash
		l.remove(e)
	}
	return e.Value
}

// PushFront inserts a new element e with value v at the front of list l and returns e.
func (l *List[T]) PushFront(v T) *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	return l.insertValue(v, l.root)
}

// PushBack inserts a new element e with value v at the back of list l and returns e.
func (l *List[T]) PushBack(v T) *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	var at *Element[T]
	if l.root != nil {
		at = l.root.prev
	}
	return l.insertValue(v, at)
}

// InsertBefore inserts a new element e with value v immediately before mark and returns e.
// If mark is not an element of l, the list is not modified.
// The mark must not be nil.
func (l *List[T]) InsertBefore(v T, mark *Element[T]) *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if mark.list != l {
		return nil
	}
	// see comment in List.Remove about initialization of l
	return l.insertValue(v, mark.prev)
}

// InsertAfter inserts a new element e with value v immediately after mark and returns e.
// If mark is not an element of l, the list is not modified.
// The mark must not be nil.
func (l *List[T]) InsertAfter(v T, mark *Element[T]) *Element[T] {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if mark.list != l {
		return nil
	}
	// see comment in List.Remove about initialization of l
	return l.insertValue(v, mark)
}

// MoveToFront moves element e to the front of list l.
// If e is not an element of l, the list is not modified.
// The element must not be nil.
func (l *List[T]) MoveToFront(e *Element[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if e.list != l || (l.root != nil && l.root.next == e) {
		return
	}
	// see comment in List.Remove about initialization of l
	l.move(e, l.root)
}

// MoveToBack moves element e to the back of list l.
// If e is not an element of l, the list is not modified.
// The element must not be nil.
func (l *List[T]) MoveToBack(e *Element[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	if e.list != l || l.root.prev == e {
		return
	}
	// see comment in List.Remove about initialization of l
	l.move(e, l.root.prev)
}

// MoveBefore moves element e to its new position before mark.
// If e or mark is not an element of l, or e == mark, the list is not modified.
// The element and mark must not be nil.
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

// MoveAfter moves element e to its new position after mark.
// If e or mark is not an element of l, or e == mark, the list is not modified.
// The element and mark must not be nil.
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

// PushBackList inserts a copy of another list at the back of list l.
// The lists l and other may be the same. They must not be nil.
func (l *List[T]) PushBackList(other *List[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	for i, e := other.Len(), other.Front(); i > 0; i, e = i-1, e.Next() {
		l.insertValue(e.Value, l.root.prev)
	}
}

// PushFrontList inserts a copy of another list at the front of list l.
// The lists l and other may be the same. They must not be nil.
func (l *List[T]) PushFrontList(other *List[T]) {
	if l.isGuarded {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	for i, e := other.Len(), other.Back(); i > 0; i, e = i-1, e.Prev() {
		l.insertValue(e.Value, l.root)
	}
}
