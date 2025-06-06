package list

import (
	"sync"
	"sync/atomic"
)

// Element — элемент generic-двусвязного списка.
type Element[T any] struct {
	next, prev *Element[T]
	list       *List[T]
	Value      T
}

// Next возвращает следующий элемент или nil.
func (e *Element[T]) Next() *Element[T] {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// Prev возвращает предыдущий элемент или nil.
func (e *Element[T]) Prev() *Element[T] {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// List — потокобезопасный generic-двусвязный список.
type List[T any] struct {
	root     Element[T]   // sentinel
	length   int64        // atomic
	mu       sync.RWMutex // мьютекс для всей структуры
	elemPool *sync.Pool   // пул для элементов
}

// New создает новый список.
func New[T any]() *List[T] {
	l := &List[T]{
		elemPool: &sync.Pool{
			New: func() any {
				return &Element[T]{}
			},
		},
	}
	l.Init()
	return l
}

// Init сбрасывает или инициализирует список.
func (l *List[T]) Init() *List[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.root.next = &l.root
	l.root.prev = &l.root
	atomic.StoreInt64(&l.length, 0)
	return l
}

// Len возвращает длину.
func (l *List[T]) Len() int {
	return int(atomic.LoadInt64(&l.length))
}

// Front возвращает первый элемент или nil.
func (l *List[T]) Front() *Element[T] {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.Len() == 0 {
		return nil
	}
	return l.root.next
}

// Back возвращает последний элемент или nil.
func (l *List[T]) Back() *Element[T] {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.Len() == 0 {
		return nil
	}
	return l.root.prev
}

// внутренний insert, НЕ делать публичным
func (l *List[T]) insert(e, at *Element[T]) *Element[T] {
	e.prev = at
	e.next = at.next
	at.next.prev = e
	at.next = e
	e.list = l
	atomic.AddInt64(&l.length, 1)
	return e
}

// внутренний remove
func (l *List[T]) remove(e *Element[T]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	atomic.AddInt64(&l.length, -1)
	// помещаем обратно в пул
	l.elemPool.Put(e)
}

// move перемещает e после at (или at == &l.root для MoveToFront)
func (l *List[T]) move(e, at *Element[T]) {
	if e == at {
		return
	}
	// Сначала удаляем из текущей позиции
	e.prev.next = e.next
	e.next.prev = e.prev

	// Потом вставляем после at
	e.prev = at
	e.next = at.next
	at.next.prev = e
	at.next = e
}

// Remove удаляет элемент, возвращает значение.
func (l *List[T]) Remove(e *Element[T]) (v T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list == l {
		v = e.Value
		l.remove(e)
	}
	return
}

// PushFront вставляет элемент в начало.
func (l *List[T]) PushFront(v T) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	elem := l.elemPool.Get().(*Element[T])
	*elem = Element[T]{Value: v}
	return l.insert(elem, &l.root)
}

// PushBack вставляет элемент в конец.
func (l *List[T]) PushBack(v T) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	elem := l.elemPool.Get().(*Element[T])
	*elem = Element[T]{Value: v}
	return l.insert(elem, l.root.prev)
}

// MoveToFront переносит e в начало.
func (l *List[T]) MoveToFront(e *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list != l || l.root.next == e {
		return
	}
	l.move(e, &l.root)
}

// MoveToBack переносит e в конец.
func (l *List[T]) MoveToBack(e *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list != l || l.root.prev == e {
		return
	}
	l.move(e, l.root.prev)
}

// SwapValues обмен значениями между двумя элементами.
func (l *List[T]) SwapValues(e1, e2 *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e1.Value, e2.Value = e2.Value, e1.Value
}

// InsertAfter вставляет новый элемент после mark.
func (l *List[T]) InsertAfter(v T, mark *Element[T]) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	if mark.list != l {
		return nil
	}
	elem := l.elemPool.Get().(*Element[T])
	*elem = Element[T]{Value: v}
	return l.insert(elem, mark)
}

// InsertBefore вставляет новый элемент перед mark.
func (l *List[T]) InsertBefore(v T, mark *Element[T]) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	if mark.list != l {
		return nil
	}
	elem := l.elemPool.Get().(*Element[T])
	*elem = Element[T]{Value: v}
	return l.insert(elem, mark.prev)
}

// MoveBefore перемещает e перед mark.
func (l *List[T]) MoveBefore(e, mark *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.prev)
}

// MoveAfter перемещает e после mark.
func (l *List[T]) MoveAfter(e, mark *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark)
}

// PushBackList — вставить копию списка other в конец.
func (l *List[T]) PushBackList(other *List[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for e := other.Front(); e != nil; e = e.Next() {
		elem := l.elemPool.Get().(*Element[T])
		*elem = Element[T]{Value: e.Value}
		l.insert(elem, l.root.prev)
	}
}

// PushFrontList — вставить копию списка other в начало.
func (l *List[T]) PushFrontList(other *List[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for e := other.Back(); e != nil; e = e.Prev() {
		elem := l.elemPool.Get().(*Element[T])
		*elem = Element[T]{Value: e.Value}
		l.insert(elem, &l.root)
	}
}
