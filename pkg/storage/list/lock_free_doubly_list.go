package list

import (
	synced "github.com/Borislavv/traefik-http-cache-plugin/pkg/sync"
	"sync/atomic"
)

type LockFreeDoublyLinkedListElement[T any] struct {
	Value T
	next  *LockFreeDoublyLinkedListElement[T]
	prev  *LockFreeDoublyLinkedListElement[T]
}

// LockFreeDoublyLinkedList is a lock-free doubly-linked list.
type LockFreeDoublyLinkedList[T any] struct {
	len  int64
	head *atomic.Pointer[LockFreeDoublyLinkedListElement[T]]
	tail *LockFreeDoublyLinkedListElement[T] // Only accessed by the single reader (event loop).
	pool *synced.BatchPool[*LockFreeDoublyLinkedListElement[T]]
}

// NewLockFreeDoublyLinkedList creates a new LockFreeDoublyLinkedList.
func NewLockFreeDoublyLinkedList[T any]() *LockFreeDoublyLinkedList[T] {
	return &LockFreeDoublyLinkedList[T]{
		head: &atomic.Pointer[LockFreeDoublyLinkedListElement[T]]{},
		pool: synced.NewBatchPool[*LockFreeDoublyLinkedListElement[T]](synced.PreallocationBatchSize, func() *LockFreeDoublyLinkedListElement[T] {
			return new(LockFreeDoublyLinkedListElement[T])
		}),
	}
}

func (l *LockFreeDoublyLinkedList[T]) Len() int64 {
	return atomic.LoadInt64(&l.len)
}

// PushFront adds a Spoke to the front of the list (lock-free).
func (l *LockFreeDoublyLinkedList[T]) PushFront(value T) *LockFreeDoublyLinkedListElement[T] {
	atomic.AddInt64(&l.len, 1)

	newElem := l.pool.Get()
	newElem.Value = value
	newElem.prev = nil
	newElem.next = nil

	for {
		currentHead := l.head.Load()
		newElem.next = currentHead
		if currentHead != nil {
			currentHead.prev = newElem
		} else {
			l.tail = newElem
		}

		if l.head.CompareAndSwap(currentHead, newElem) {
			return newElem
		}
	}
}

// MoveToFront moves an element to the front of the list (lock-free).
func (l *LockFreeDoublyLinkedList[T]) MoveToFront(e *LockFreeDoublyLinkedListElement[T]) {
	if e == nil {
		return
	}

	// Step 1: Unlink the node (lock-free).
	for {
		next := e.next
		prev := e.prev

		if prev != nil {
			prev.next = next
		} else if !l.head.CompareAndSwap(e, next) {
			continue // Retry if head was changed.
		}

		if next != nil {
			next.prev = prev
		} else {
			l.tail = prev // Update tail if moving the last node.
		}

		// Step 2: Prepend to head (like PushFront).
		for {
			currentHead := l.head.Load()
			e.next = currentHead
			e.prev = nil
			if currentHead != nil {
				currentHead.prev = e
			} else {
				l.tail = e
			}

			if l.head.CompareAndSwap(currentHead, e) {
				break
			}
		}
		break
	}
}

// Remove removes an element from the list (lock-free).
func (l *LockFreeDoublyLinkedList[T]) Remove(e *LockFreeDoublyLinkedListElement[T]) {
	if e == nil {
		return
	}

	defer func() {
		l.pool.Put(e)
		atomic.AddInt64(&l.len, -1)
	}()

	for {
		next := e.next
		prev := e.prev

		if prev != nil {
			prev.next = next
		} else if !l.head.CompareAndSwap(e, next) {
			continue // Retry if head was changed.
		}

		if next != nil {
			next.prev = prev
		} else {
			l.tail = prev // Update tail if removing the last node.
		}
		break
	}
}

// Back returns the last element (safe for single reader).
func (l *LockFreeDoublyLinkedList[T]) Back() *LockFreeDoublyLinkedListElement[T] {
	return l.tail
}

// Next returns the next element (safe for single reader).
func (e *LockFreeDoublyLinkedListElement[T]) Next() *LockFreeDoublyLinkedListElement[T] {
	return e.next
}
