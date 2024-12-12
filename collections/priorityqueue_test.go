package collections

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flanksource/commons/test/matchers"
	. "github.com/onsi/gomega"
)

func TestPriorityQueueString(t *testing.T) {
	g := NewWithT(t)

	pq, err := NewQueue(QueueOpts[string]{
		Comparator: strings.Compare,
		Metrics: MetricsOpts[string]{
			Labels: map[string]any{
				"const_label": "value1",
			},
			Labeller: map[string]func(i string) string{
				"prefix": func(i string) string {
					if len(i) == 0 {
						return i
					}
					return i[0:1]
				},
			},
		},
	})

	g.Expect(err).To(BeNil())
	g.Expect(pq.Size()).To(BeNumerically("==", 0))

	pq.Enqueue("item1")
	pq.Enqueue("batch1")

	g.Expect(pq.Size()).To(BeNumerically("==", 2))
	g.Expect(first(pq.Peek())).To(Equal("batch1"))
	g.Expect(first(pq.Dequeue())).To(Equal("batch1"))
	g.Expect(pq.Size()).To(BeNumerically("==", 1))
	g.Expect(first(pq.Dequeue())).Should(Equal("item1"))
	g.Expect(first(pq.Peek())).To(Equal(""))

	// g.Expect(dumpMetrics("priority")).Should(ContainSubstring("zz"))

	g.Expect("priority_queue_enqueued_total").To(matchers.MatchCounter(1, "prefix", "i"))
	g.Expect("priority_queue_enqueued_total").To(matchers.MatchCounter(1, "prefix", "i"))
	g.Expect("priority_queue_dequeued_total").To(matchers.MatchCounter(1, "prefix", "i"))
	g.Expect("priority_queue_duration_count").To(matchers.MatchCounter(1, "prefix", "i"))

	g.Expect("priority_queue_size").To(matchers.MatchCounter(0))
}

type QueueItem struct {
	Timestamp time.Time // Queued time
	Obj       map[string]any
}

func (t *QueueItem) Name() string {
	return t.Obj["name"].(string)
}

func NewQueueItem(obj map[string]any) *QueueItem {
	return &QueueItem{
		Timestamp: time.Now(),
		Obj:       obj,
	}
}

func TestPriorityQueue(t *testing.T) {
	g := NewWithT(t)

	pq, err := NewQueue(QueueOpts[*QueueItem]{
		Metrics: MetricsOpts[*QueueItem]{
			Labels: map[string]any{
				"const_label": "value1",
			},
			Labeller: map[string]func(i *QueueItem) string{
				"prefix": func(i *QueueItem) string {
					return "dummy"
				},
			},
		},
		Comparator: func(a, b *QueueItem) int {
			return strings.Compare(a.Obj["name"].(string), b.Obj["name"].(string))
		},
		Dedupe: true,
		Equals: func(a, b *QueueItem) bool {
			return strings.EqualFold(a.Obj["name"].(string), b.Obj["name"].(string))
		},
	})
	g.Expect(err).To(BeNil())
	g.Expect(pq.Size()).To(BeZero())

	names := []string{"bob", "foo", "bar", "eve", "baz", "alice", "bob"}
	for _, name := range names {
		pq.Enqueue(NewQueueItem(map[string]any{"name": name}))
	}

	g.Expect(pq.Size()).To(BeNumerically("==", len(names)))

	expected := []string{"alice", "bar", "baz", "bob", "eve", "foo"}
	for _, e := range expected {
		g.Expect(first(pq.Peek()).Name()).To(Equal(e))
		g.Expect(first(pq.Dequeue()).Name()).Should(Equal(e))
	}

	g.Expect(pq.Size()).To(BeZero())
}

func TestPriorityQueueDedupe(t *testing.T) {
	g := NewWithT(t)

	pq, err := NewQueue(QueueOpts[string]{
		Equals:     func(a, b string) bool { return a == b },
		Dedupe:     true,
		Comparator: strings.Compare,
		Metrics: MetricsOpts[string]{
			Name: "dedupe_queue",
		},
	})

	g.Expect(err).To(BeNil())
	g.Expect(pq.Size()).To(BeNumerically("==", 0))

	pq.Enqueue("item1")
	pq.Enqueue("batch1")

	g.Expect(pq.Size()).To(BeNumerically("==", 2))
	pq.Enqueue("item1")
	pq.Enqueue("batch2")
	g.Expect(pq.Size()).To(BeNumerically("==", 4))

	pq.Dequeue() // batch1
	g.Expect(pq.Size()).To(BeNumerically("==", 3))
	pq.Dequeue() // batch2
	g.Expect(pq.Size()).To(BeNumerically("==", 2))

	pq.Dequeue() // item1
	g.Expect(pq.Size()).To(BeNumerically("==", 0))

	g.Expect("dedupe_queue_enqueued_total").To(matchers.MatchCounter(4))
	g.Expect("dedupe_queue_dequeued_total").To(matchers.MatchCounter(3))
	g.Expect("dedupe_queue_deduped_total").To(matchers.MatchCounter(1))
	g.Expect("dedupe_queue_size").To(matchers.MatchCounter(0))
}

func TestPriorityQueueConcurrency(t *testing.T) {
	g := NewWithT(t)
	pq, err := NewQueue(QueueOpts[string]{
		Comparator: strings.Compare,
		Metrics: MetricsOpts[string]{
			Name: "concurrent_queue",
		},
	})
	g.Expect(err).To(BeNil())

	const numGoroutines = 50
	const itemsPerGoroutine = 100

	var wg sync.WaitGroup

	start := time.Now()
	// Start producer goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				pq.Enqueue(fmt.Sprintf("item-%d-%d", id, j))
			}
		}(i)
	}

	// wg.Wait()
	// g.Expect(time.Since(start).Milliseconds()).To(BeNumerically("<", 100))
	// g.Expect(pq.Size()).To(BeNumerically("==", itemsPerGoroutine*numGoroutines))
	// start = time.Now()
	// Start consumer goroutines
	expectedCount := numGoroutines * itemsPerGoroutine
	dequeued := atomic.Int32{}
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dequeued.Load() < int32(expectedCount) {
				if _, ok := pq.Dequeue(); ok {
					dequeued.Add(1)
				} else {
					time.Sleep(1 * time.Millisecond)
				}
			}
		}()
	}

	// Wait for all operations to complete
	wg.Wait()
	g.Expect(time.Since(start).Milliseconds()).To(BeNumerically("<", 100))
	g.Expect(int(dequeued.Load())).To(Equal(expectedCount))
	g.Expect("concurrent_queue_duration_count").To(matchers.MatchCounter(int64(expectedCount)))

	g.Expect("concurrent_queue_size").To(matchers.MatchCounter(0))
	g.Expect(pq.Size()).To(BeNumerically("==", 0))

	t.Log("\n" + matchers.DumpMetrics("priority"))
}

func first[T1 any, T2 any](a T1, _ T2) T1 {
	return a
}
