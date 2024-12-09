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

func TestPriorityQueue(t *testing.T) {
	g := NewWithT(t)

	pq, err := New(QueueOpts[string]{
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
		}})

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

func TestPriorityQueueDedupe(t *testing.T) {
	g := NewWithT(t)

	pq, err := New(QueueOpts[string]{
		Equals:     func(a, b string) bool { return a == b },
		Dedupe:     true,
		Comparator: strings.Compare,
		Metrics: MetricsOpts[string]{
			MetricName: "dedupe_queue",
		}})

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
	pq, err := New(QueueOpts[string]{
		Comparator: strings.Compare,
	})
	Expect(err).To(BeNil())

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

	// g.Expect(matchers.DumpMetrics("priority")).To(ContainSubstring("zz"))
	g.Expect(int(dequeued.Load())).To(Equal(expectedCount))
	// g.Expect("priority_queue_duration_sum").To(matchers.MatchCounter(1))
	g.Expect("priority_queue_duration_count").To(matchers.MatchCounter(int64(expectedCount)))

	g.Expect("priority_queue_size").To(matchers.MatchCounter(0))
	g.Expect(pq.Size()).To(BeNumerically("==", 0))

	t.Log("\n" + matchers.DumpMetrics("priority"))

}

func first[T1 any, T2 any](a T1, b T2) T1 {
	return a
}
func second[T1 any, T2 any](a T1, b T2) T2 {
	return b
}
