// Package prommetrics opts priority queues into Prometheus metrics.
package prommetrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sync"
	"time"
)

var (
	metricCache     = map[string]any{}
	metricCacheLock sync.Mutex
)

// Options contains options for queue metrics
type Options[T comparable] struct {
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
	opts          Options[T]
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

func (m *metrics[T]) Enqueue(value T, currentSize int) {
	if m.opts.Disable {
		return
	}
	labels := m.labels(value)
	m.enqueuedTotal.With(labels).Inc()
	m.queueSize.Set(float64(currentSize))
}

func (m *metrics[T]) Dedupe(value T, currentSize int) {
	if m.opts.Disable {
		return
	}
	labels := m.labels(value)
	m.dedupedTotal.With(labels).Inc()
	m.queueSize.Set(float64(currentSize))
}

func (m *metrics[T]) Dequeue(item T, elapsed time.Duration, currentSize int) {
	if m.opts.Disable {
		return
	}
	labels := m.labels(item)
	m.dequeuedTotal.With(labels).Inc()
	m.queueDuration.With(labels).Observe(float64(elapsed.Milliseconds()))
	m.queueSize.Set(float64(currentSize))
}

func New[T comparable](opts Options[T]) *metrics[T] {
	keys := make([]string, 0, len(opts.Labels))
	for key := range opts.Labels {
		keys = append(keys, key)
	}
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

	metricCacheLock.Lock()
	defer metricCacheLock.Unlock()

	return &metrics[T]{
		opts:          opts,
		enqueuedTotal: getOrCreateCounterVec(opts.Name, "enqueued_total", "The total number of enqueued items", keys),
		dedupedTotal:  getOrCreateCounterVec(opts.Name, "deduped_total", "The total number of deduped items", keys),
		dequeuedTotal: getOrCreateCounterVec(opts.Name, "dequeued_total", "The total number of dequeued items", keys),
		queueSize:     getOrCreateGauge(opts.Name, "size", "The current size of the queue", labels),
		queueDuration: getOrCreateHistogramVec(opts.Name, "duration", "Time an object spent in the queue in milliseconds", keys, opts.DurationBuckets),
	}
}

func getOrCreateCounterVec(prefix, suffix, help string, keys []string) *prometheus.CounterVec {
	name := fmt.Sprintf("%s_%s", prefix, suffix)
	if val, ok := metricCache[name]; ok {
		return val.(*prometheus.CounterVec)
	}

	counter := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, keys)
	metricCache[name] = counter
	return counter
}

func getOrCreateGauge(prefix, suffix, help string, labels prometheus.Labels) prometheus.Gauge {
	name := fmt.Sprintf("%s_%s", prefix, suffix)
	if val, ok := metricCache[name]; ok {
		return val.(prometheus.Gauge)
	}

	gauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        name,
		Help:        help,
		ConstLabels: labels,
	})
	metricCache[name] = gauge
	return gauge
}

func getOrCreateHistogramVec(prefix, suffix, help string, keys []string, buckets []float64) *prometheus.HistogramVec {
	name := fmt.Sprintf("%s_%s", prefix, suffix)
	if val, ok := metricCache[name]; ok {
		return val.(*prometheus.HistogramVec)
	}

	histogram := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}, keys)
	metricCache[name] = histogram
	return histogram
}

func (m *metrics[T]) SetSize(size int) {
	if !m.opts.Disable {
		m.queueSize.Set(float64(size))
	}
}
