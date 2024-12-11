// Copyright (c) 2015, Emir Pasic. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package priorityqueue implements a priority queue backed by binary queue.
//
// A thread-safe priority queue based on a priority queue.
// The elements of the priority queue are ordered by a comparator provided at queue construction time.
//
// The heap of this queue is the least/smallest element with respect to the specified ordering.
// If multiple elements are tied for least value, the heap is one of those elements arbitrarily.
//
// Structure is thread safe.
//
// References: https://en.wikipedia.org/wiki/Priority_queue
package collections

import (
	"errors"
	"fmt"
	"iter"
	"strings"
	"sync"
	"time"

	"github.com/emirpasic/gods/v2/queues"
	"github.com/emirpasic/gods/v2/trees/binaryheap"
	"github.com/emirpasic/gods/v2/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/samber/lo"
)

// MetricsOpts contains options for queue metrics
type MetricsOpts[T comparable] struct {
	Labels          map[string]any
	Labeller        map[string]func(i T) string
	DurationBuckets []float64
	Name            string
	Disable         bool
}

type metrics[T comparable] struct {
	enqueuedTotal *prometheus.CounterVec
	dequeuedTotal *prometheus.CounterVec
	dedupedTotal  *prometheus.CounterVec
	queueSize     prometheus.Gauge
	queueDuration *prometheus.HistogramVec
	opts          MetricsOpts[T]
}

func (m *metrics[T]) labels(item T) map[string]string {
	labels := prometheus.Labels{}
	for k, v := range m.opts.Labels {
		labels[k] = fmt.Sprintf("%v", v)
	}
	for k, v := range m.opts.Labeller {
		o := v(item)
		labels[k] = o
	}
	return labels
}

func (m *metrics[T]) enqueue(value T, currentSize int) {
	if m.opts.Disable {
		return
	}
	labels := m.labels(value)
	m.enqueuedTotal.With(labels).Inc()
	m.queueSize.Set(float64(currentSize))
}

func (m *metrics[T]) dedupe(value T, currentSize int) {
	if m.opts.Disable {
		return
	}
	labels := m.labels(value)
	m.dedupedTotal.With(labels).Inc()
	m.queueSize.Set(float64(currentSize))
}

func (m *metrics[T]) dequeue(item queueItem[T], currentSize int) {
	if m.opts.Disable {
		return
	}
	labels := m.labels(item.item)
	m.dequeuedTotal.With(labels).Inc()
	m.queueDuration.With(labels).Observe(float64(time.Since(item.inserted).Milliseconds()))
	m.queueSize.Set(float64(currentSize))

}

func newMetrics[T comparable](opts MetricsOpts[T]) *metrics[T] {

	keys := lo.Keys(opts.Labels)

	labels := prometheus.Labels{}

	for k, v := range opts.Labels {
		labels[k] = fmt.Sprintf("%v", v)
	}

	for k := range opts.Labeller {
		keys = append(keys, k)
	}
	if len(opts.DurationBuckets) == 0 {
		opts.DurationBuckets = []float64{
			1, 10, 50, 100, 500, 1000, 3 * 1000, 10 * 1000, 30 * 1000, 60 * 1000, 300 * 1000,
		}
	}

	if opts.Name == "" {
		opts.Name = "priority_queue"
	}

	return &metrics[T]{
		opts: opts,
		enqueuedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: opts.Name + "_enqueued_total",
			Help: "The total number of enqueued items",
		}, keys),
		dedupedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: opts.Name + "_deduped_total",
			Help: "The total number of enqueued items",
		}, keys),
		dequeuedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: opts.Name + "_dequeued_total",
			Help: "The total number of dequeued items",
		}, keys),
		queueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name:        opts.Name + "_size",
			Help:        "The current size of the queue",
			ConstLabels: labels,
		}),
		queueDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    opts.Name + "_duration",
			Help:    "Time an object spent in the queue in milliseconds",
			Buckets: opts.DurationBuckets,
		}, keys),
	}
}

type queueItem[T comparable] struct {
	item     T
	inserted time.Time
}

// Assert Queue implementation
var _ queues.Queue[int] = (*Queue[int])(nil)

// Queue holds elements in an array-list
type Queue[T comparable] struct {
	heap       *binaryheap.Heap[queueItem[T]]
	Comparator utils.Comparator[T]
	Equals     func(a, b T) bool
	metrics    *metrics[T]
	mutex      sync.RWMutex
	Dedupe     bool
}

type QueueOpts[T comparable] struct {
	Comparator utils.Comparator[T]
	Dedupe     bool
	Equals     func(a, b T) bool
	Metrics    MetricsOpts[T]
}

func NewQueue[T comparable](opts QueueOpts[T]) (*Queue[T], error) {
	if opts.Dedupe && opts.Equals == nil {
		return nil, errors.New("dedupe requires Equals function")
	}

	if opts.Comparator == nil {
		return nil, errors.New("a comparator function is required")
	}

	return &Queue[T]{
		heap: binaryheap.NewWith(func(a, b queueItem[T]) int {
			return opts.Comparator(a.item, b.item)
		}),
		Comparator: opts.Comparator,
		Equals:     opts.Equals,
		metrics:    newMetrics(opts.Metrics),
		Dedupe:     opts.Dedupe,
	}, nil
}

// Enqueue adds a value to the end of the queue
func (queue *Queue[T]) Enqueue(value T) {
	queue.mutex.Lock()
	queue.heap.Push(queueItem[T]{
		item:     value,
		inserted: time.Now(),
	})
	queue.metrics.enqueue(value, queue.heap.Size())
	queue.mutex.Unlock()
}

type Equals[T any] interface {
	Equals(T) bool
}

// Dequeue removes first element of the queue and returns it, or nil if queue is empty.
// Second return parameter is true, unless the queue was empty and there was nothing to dequeue.
func (queue *Queue[T]) Dequeue() (T, bool) {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	wrapper, ok := queue.heap.Pop()
	if !ok {
		var zero T
		return zero, false
	}

	queue.metrics.dequeue(wrapper, queue.heap.Size())

	if queue.Dedupe {

		// Keep dequeuing while next item is the same as current
		for {
			next, hasNext := queue.heap.Peek()
			if !hasNext {
				break
			}

			if queue.Equals(next.item, wrapper.item) {
				queue.heap.Pop()
				queue.metrics.dedupe(next.item, queue.heap.Size())
			} else {
				break
			}
		}
	}

	return wrapper.item, true
}

// Peek returns top element on the queue without removing it, or nil if queue is empty.
// Second return parameter is true, unless the queue was empty and there was nothing to peek.
func (queue *Queue[T]) Peek() (value T, ok bool) {
	queue.mutex.RLock()
	defer queue.mutex.RUnlock()
	wrapper, ok := queue.heap.Peek()
	return wrapper.item, ok
}

// Empty returns true if queue does not contain any elements.
func (queue *Queue[T]) Empty() bool {
	queue.mutex.RLock()
	defer queue.mutex.RUnlock()
	return queue.heap.Empty()
}

// Size returns number of elements within the queue.
func (queue *Queue[T]) Size() int {
	queue.mutex.RLock()
	defer queue.mutex.RUnlock()
	return queue.heap.Size()
}

// Clear removes all elements from the queue.
func (queue *Queue[T]) Clear() {
	queue.mutex.Lock()
	queue.heap.Clear()
	queue.mutex.Unlock()

	queue.metrics.queueSize.Set(0)
}

// Values returns all elements in the queue.
func (queue *Queue[T]) Values() []T {
	queue.mutex.RLock()
	defer queue.mutex.RUnlock()
	values := make([]T, queue.heap.Size())
	for it := queue.heap.Iterator(); it.Next(); {
		values[it.Index()] = it.Value().item
	}
	return values
}

// String returns a string representation of container
func (queue *Queue[T]) String() string {
	queue.mutex.RLock()
	defer queue.mutex.RUnlock()
	str := "PriorityQueue\n"
	values := make([]string, queue.heap.Size())
	for index, value := range queue.heap.Values() {
		values[index] = fmt.Sprintf("%v", value)
	}
	str += strings.Join(values, ", ")
	return str
}

// Iterator returns a new iterator for the queue.
func (queue *Queue[T]) Iterator() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			v, ok := queue.Dequeue()
			if ok {
				yield(v)
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}
