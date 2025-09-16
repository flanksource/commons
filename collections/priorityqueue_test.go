package collections

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flanksource/commons/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PriorityQueue", func() {
	Describe("String queue", func() {
		It("should handle string priority queue operations", func() {
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

			Expect(err).To(BeNil())
			Expect(pq.Size()).To(BeNumerically("==", 0))

			pq.Enqueue("item1")
			pq.Enqueue("batch1")

			Expect(pq.Size()).To(BeNumerically("==", 2))
			Expect(first(pq.Peek())).To(Equal("batch1"))
			Expect(first(pq.Dequeue())).To(Equal("batch1"))
			Expect(pq.Size()).To(BeNumerically("==", 1))
			Expect(first(pq.Dequeue())).Should(Equal("item1"))
			Expect(first(pq.Peek())).To(Equal(""))

			Expect("priority_queue_enqueued_total").To(matchers.MatchCounter(1, "prefix", "i"))
			Expect("priority_queue_enqueued_total").To(matchers.MatchCounter(1, "prefix", "i"))
			Expect("priority_queue_dequeued_total").To(matchers.MatchCounter(1, "prefix", "i"))
			Expect("priority_queue_duration_count").To(matchers.MatchCounter(1, "prefix", "i"))

			Expect("priority_queue_size").To(matchers.MatchCounter(0))
		})
	})

	Describe("Object queue", func() {
		It("should handle object priority queue operations with deduplication", func() {
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
			Expect(err).To(BeNil())
			Expect(pq.Size()).To(BeZero())

			names := []string{"bob", "foo", "bar", "eve", "baz", "alice", "bob"}
			for _, name := range names {
				pq.Enqueue(NewQueueItem(map[string]any{"name": name}))
			}

			Expect(pq.Size()).To(BeNumerically("==", len(names)))

			expected := []string{"alice", "bar", "baz", "bob", "eve", "foo"}
			for _, e := range expected {
				Expect(first(pq.Peek()).Name()).To(Equal(e))
				Expect(first(pq.Dequeue()).Name()).Should(Equal(e))
			}

			Expect(pq.Size()).To(BeZero())
		})
	})

	Describe("Deduplication", func() {
		It("should handle deduplication correctly", func() {
			pq, err := NewQueue(QueueOpts[string]{
				Equals:     func(a, b string) bool { return a == b },
				Dedupe:     true,
				Comparator: strings.Compare,
				Metrics: MetricsOpts[string]{
					Name: "dedupe_queue",
				},
			})

			Expect(err).To(BeNil())
			Expect(pq.Size()).To(BeNumerically("==", 0))

			pq.Enqueue("item1")
			pq.Enqueue("batch1")

			Expect(pq.Size()).To(BeNumerically("==", 2))
			pq.Enqueue("item1")
			pq.Enqueue("batch2")
			Expect(pq.Size()).To(BeNumerically("==", 4))

			pq.Dequeue() // batch1
			Expect(pq.Size()).To(BeNumerically("==", 3))
			pq.Dequeue() // batch2
			Expect(pq.Size()).To(BeNumerically("==", 2))

			pq.Dequeue() // item1
			Expect(pq.Size()).To(BeNumerically("==", 0))

			Expect("dedupe_queue_enqueued_total").To(matchers.MatchCounter(4))
			Expect("dedupe_queue_dequeued_total").To(matchers.MatchCounter(3))
			Expect("dedupe_queue_deduped_total").To(matchers.MatchCounter(1))
			Expect("dedupe_queue_size").To(matchers.MatchCounter(0))
		})
	})

	Describe("Concurrency", func() {
		It("should handle concurrent operations safely", func() {
			pq, err := NewQueue(QueueOpts[string]{
				Comparator: strings.Compare,
				Metrics: MetricsOpts[string]{
					Name: "concurrent_queue",
				},
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
			Expect(time.Since(start).Milliseconds()).To(BeNumerically("<", 100))
			Expect(int(dequeued.Load())).To(Equal(expectedCount))
			Expect("concurrent_queue_duration_count").To(matchers.MatchCounter(int64(expectedCount)))

			Expect("concurrent_queue_size").To(matchers.MatchCounter(0))
			Expect(pq.Size()).To(BeNumerically("==", 0))

			GinkgoWriter.Printf("\n" + matchers.DumpMetrics("priority"))
		})
	})

	Describe("Delayed enqueuing", func() {
		It("should handle delayed items correctly", func() {
			pq, err := NewQueue(QueueOpts[string]{
				Equals:     func(a, b string) bool { return a == b },
				Comparator: strings.Compare,
				Metrics: MetricsOpts[string]{
					Disable: true,
				},
			})

			Expect(err).To(BeNil())
			Expect(pq.Size()).To(BeNumerically("==", 0))

			// Enqueue items with different delays: 3s, 500ms, and immediate
			pq.EnqueueWithDelay("item1-delay-3s", 3*time.Second)
			pq.EnqueueWithDelay("item1-delay-500ms", 500*time.Millisecond)
			pq.Enqueue("item1-immediate")

			Expect(pq.Size()).To(BeNumerically("==", 3))

			// Immediate item should be available right away
			item, ok := pq.Dequeue()
			Expect(ok).To(BeTrue())
			Expect(item).To(Equal("item1-immediate"))

			// 500ms delayed item should not be available immediately
			item, ok = pq.Dequeue()
			Expect(ok).To(BeFalse())
			Expect(item).To(BeEmpty())

			// After 600ms, the 500ms delayed item should be available
			Eventually(func() string {
				item, ok := pq.Dequeue()
				if !ok {
					return ""
				}
				return item
			}, 800*time.Millisecond, 50*time.Millisecond).Should(Equal("item1-delay-500ms"))

			// 3s delayed item should not be available yet
			Consistently(func() bool {
				_, ok := pq.Dequeue()
				return ok
			}, 1*time.Second, 100*time.Millisecond).Should(BeFalse())

			// After 3.5s total, the 3s delayed item should be available
			Eventually(func() string {
				item, ok := pq.Dequeue()
				if !ok {
					return ""
				}
				return item
			}, 3*time.Second, 100*time.Millisecond).Should(Equal("item1-delay-3s"))

			// Queue should now be empty
			Expect(pq.Size()).To(BeZero())
			item, ok = pq.Dequeue()
			Expect(ok).To(BeFalse())
			Expect(item).To(BeEmpty())
		})
	})
})

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

func first[T1 any, T2 any](a T1, _ T2) T1 {
	return a
}
