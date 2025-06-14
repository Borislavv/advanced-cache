package list

import (
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"sync"
)

// Element is a node in the doubly linked list that holds a value of type T.
// Never touch fields directly outside of List methods.
type Element[T Sortable] struct {
	next, prev *Element[T]
	list       *List[T]
	value      T
}

// List returns the whole list if this element.
func (e *Element[T]) List() *List[T] {
	return e.list
}

// Value returns the value. NOT thread-safe! Only use inside locked section or single-threaded use!
func (e *Element[T]) Value() T {
	return e.value
}

type Sortable interface {
	Weight() uintptr
}

// List is a generic doubly linked list with optional thread safety.
type List[T Sortable] struct {
	len      int
	mu       *sync.Mutex
	root     *Element[T]
	elemPool *synced.BatchPool[*Element[T]]
}

// New creates a new list. If isThreadSafe is true, all ops are guarded by a mutex.
func New[T Sortable]() *List[T] {
	l := &List[T]{
		mu: &sync.Mutex{},
		elemPool: synced.NewBatchPool[*Element[T]](synced.PreallocateBatchSize, func() *Element[T] {
			return &Element[T]{}
		}),
	}
	l.init()
	return l
}

func (l *List[T]) init() *List[T] {
	root := l.elemPool.Get()
	*root = Element[T]{}
	l.root = root
	l.root.next = l.root
	l.root.prev = l.root
	l.len = 0
	return l
}

// Len returns the list length (O(1)). Thread-safe if guarded.
func (l *List[T]) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.len
}

func (l *List[T]) insert(e, at *Element[T]) *Element[T] {
	e.prev = at
	e.next = at.next
	at.next.prev = e
	at.next = e
	e.list = l
	l.len++
	return e
}

func (l *List[T]) insertValue(v T, at *Element[T]) *Element[T] {
	el := l.elemPool.Get()
	*el = Element[T]{value: v}
	return l.insert(el, at)
}

func (l *List[T]) remove(e *Element[T]) T {
	e.prev.next = e.next
	e.next.prev = e.prev
	val := e.value
	e.next = nil
	e.prev = nil
	e.list = nil
	l.elemPool.Put(e)
	l.len--
	return val
}

// Remove removes e from l and returns its value. Thread-safe.
func (l *List[T]) Remove(e *Element[T]) T {
	l.mu.Lock()
	defer l.mu.Unlock()

	if e == nil || e.list != l {
		var zero T
		return zero
	}
	return l.remove(e)
}

// PushFront inserts v at the front and returns new element. Thread-safe if guarded.
func (l *List[T]) PushFront(v T) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.insertValue(v, l.root)
}

// PushBack inserts v at the back and returns new element. Thread-safe if guarded.
func (l *List[T]) PushBack(v T) *Element[T] {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.insertValue(v, l.root.prev)
}

// MoveToFront moves e to the front of the list without removing it from memory or touching the pool.
func (l *List[T]) MoveToFront(e *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if e == nil || e.list != l || e == l.root.next {
		return
	}

	// Detach e
	e.prev.next = e.next
	e.next.prev = e.prev

	// Move right after root (to front)
	e.prev = l.root
	e.next = l.root.next
	l.root.next.prev = e
	l.root.next = e
}

// SwapElements moves a and b (nodes, not just values) in the list. Thread-safe if guarded.
func (l *List[T]) SwapElements(a, b *Element[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if a == nil || b == nil || a.list != l || b.list != l || a == b {
		return
	}

	// Actually swap elements, not values, for safety.
	// Remove both (in either order), then re-insert each at the other's old position.
	aPrev, aNext := a.prev, a.next
	bPrev, bNext := b.prev, b.next

	// Remove both from the list
	a.prev.next = a.next
	a.next.prev = a.prev
	b.prev.next = b.next
	b.next.prev = b.prev

	// Insert a at b's original position
	a.prev = bPrev
	a.next = bNext
	bPrev.next = a
	bNext.prev = a

	// Insert b at a's original position
	b.prev = aPrev
	b.next = aNext
	aPrev.next = b
	aNext.prev = b
}

// Walk executes fn for each element in order (under lock if guarded).
func (l *List[T]) Walk(dir Direction, fn func(l *List[T], el *Element[T]) (shouldContinue bool)) {
	l.mu.Lock()
	defer l.mu.Unlock()

	switch dir {
	case FromFront:
		for e, n := l.root.next, l.len; n > 0 && e != nil; n, e = n-1, e.next {
			if !fn(l, e) {
				return
			}
		}
	case FromBack:
		for e, n := l.root.prev, l.len; n > 0 && e != nil; n, e = n-1, e.prev {
			if !fn(l, e) {
				return
			}
		}
	default:
		panic("unknown walk direction")
	}
}

func (l *List[T]) Sort(ord Order) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.len < 2 {
		return
	}

	swapped := true
	for swapped {
		swapped = false
		for curr := l.root.next; curr != nil && curr.next != l.root; curr = curr.next {
			weightA := curr.Value().Weight()
			weightB := curr.next.Value().Weight()
			if (ord == DESC && weightA < weightB) ||
				(ord == ASC && weightA > weightB) {
				l.SwapElements(curr, curr.next)
				swapped = true
			}
		}
	}
}
