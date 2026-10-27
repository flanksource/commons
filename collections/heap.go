package collections

import "container/heap"

type queueHeap[T any] struct {
	items   []T
	compare func(T, T) int
}

type heapAdapter[T any] struct{ *queueHeap[T] }

func (h heapAdapter[T]) Len() int           { return len(h.items) }
func (h heapAdapter[T]) Less(i, j int) bool { return h.compare(h.items[i], h.items[j]) < 0 }
func (h heapAdapter[T]) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h heapAdapter[T]) Push(value any)     { h.items = append(h.items, value.(T)) }
func (h heapAdapter[T]) Pop() any {
	last := len(h.items) - 1
	value := h.items[last]
	var zero T
	h.items[last] = zero
	h.items = h.items[:last]
	return value
}

func newQueueHeap[T any](compare func(T, T) int) *queueHeap[T] {
	return &queueHeap[T]{compare: compare}
}
func (h *queueHeap[T]) Push(value T) { heap.Push(heapAdapter[T]{h}, value) }
func (h *queueHeap[T]) Pop() (T, bool) {
	if len(h.items) == 0 {
		var zero T
		return zero, false
	}
	return heap.Pop(heapAdapter[T]{h}).(T), true
}
func (h *queueHeap[T]) Peek() (T, bool) {
	if len(h.items) == 0 {
		var zero T
		return zero, false
	}
	return h.items[0], true
}
func (h *queueHeap[T]) Size() int   { return len(h.items) }
func (h *queueHeap[T]) Empty() bool { return len(h.items) == 0 }
func (h *queueHeap[T]) Clear()      { h.items = nil }
func (h *queueHeap[T]) Values() []T { return h.items }
