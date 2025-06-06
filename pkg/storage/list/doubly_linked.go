package list

import (
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"sync"
)

type Element[T any] struct {
	next, prev *Element[T]
	list       *List[T]
	Value      T
}

func (e *Element[T]) Next() *Element[T] {
	if p := e.next; e.list != nil && p != e.list.root {
		return p
	}
	return nil
}

func (e *Element[T]) Prev() *Element[T] {
	if p := e.prev; e.list != nil && p != e.list.root {
		return p
	}
	return nil
}

type List[T any] struct {
	mu       sync.RWMutex
	root     *Element[T]
	elemPool *synced.BatchPool[*Element[T]]
}

// New создает новый список.
func New[T any]() *List[T] {
	l := &List[T]{
		elemPool: synced.NewBatchPool[*Element[T]](synced.PreallocationBatchSize, func() *Element[T] {
			return &Element[T]{}
		}),
	}
	l.Init()
	return l
}

func (l *List[T]) Init() *List[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.root.next = l.root
	l.root.prev = l.root
	return l
}

func (l *List[T]) Front() *Element[T] {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.root != nil {
		if next := l.root.next; next != nil {
			return next
		}
	}
	return nil
}

func (l *List[T]) Back() *Element[T] {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.root != nil {
		if prev := l.root.prev; prev != nil {
			return prev
		}
	}
	return nil
}

func (l *List[T]) insert(e, at *Element[T]) *Element[T] {
	e.prev = at
	e.next = at.next
	at.next.prev = e
	at.next = e
	e.list = l
	return e
}

func (l *List[T]) remove(e *Element[T]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.elemPool.Put(e)
}

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

func (l *List[T]) Remove(e *Element[T]) (v T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list == l {
		v = e.Value
		l.remove(e)
	}
	return
}

func (l *List[T]) PushFront(v T) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	elem := l.elemPool.Get()
	*elem = Element[T]{Value: v}
	return l.insert(elem, l.root)
}

func (l *List[T]) PushBack(v T) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	elem := l.elemPool.Get()
	*elem = Element[T]{Value: v}
	return l.insert(elem, l.root.prev)
}

func (l *List[T]) MoveToFront(e *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list != l || l.root.next == e {
		return
	}
	l.move(e, l.root)
}

func (l *List[T]) MoveToBack(e *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list != l || l.root.prev == e {
		return
	}
	l.move(e, l.root.prev)
}

func (l *List[T]) SwapValues(e1, e2 *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e1.Value, e2.Value = e2.Value, e1.Value
}

func (l *List[T]) InsertAfter(v T, mark *Element[T]) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	if mark.list != l {
		return nil
	}
	elem := l.elemPool.Get()
	*elem = Element[T]{Value: v}
	return l.insert(elem, mark)
}

func (l *List[T]) InsertBefore(v T, mark *Element[T]) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	if mark.list != l {
		return nil
	}
	elem := l.elemPool.Get()
	*elem = Element[T]{Value: v}
	return l.insert(elem, mark.prev)
}

func (l *List[T]) MoveBefore(e, mark *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.prev)
}

func (l *List[T]) MoveAfter(e, mark *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark)
}

func (l *List[T]) PushBackList(other *List[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for e := other.Front(); e != nil; e = e.Next() {
		elem := l.elemPool.Get()
		*elem = Element[T]{Value: e.Value}
		l.insert(elem, l.root.prev)
	}
}

func (l *List[T]) PushFrontList(other *List[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for e := other.Back(); e != nil; e = e.Prev() {
		elem := l.elemPool.Get()
		*elem = Element[T]{Value: e.Value}
		l.insert(elem, l.root)
	}
}
