package har_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/flanksource/commons/har"
	commonshttp "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const hangPath = "/hang"

// hangingServer accepts every request and never answers it: the handler
// blocks until the client gives up and the connection's context is done.
func hangingServer() *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	DeferCleanup(srv.Close)
	return srv
}

// fullCaptureConfig mirrors the oipa-cli test runner: every body captured.
func fullCaptureConfig() har.HARConfig {
	cfg := har.DefaultConfig()
	cfg.MaxBodySize = 0
	return cfg
}

// startHungGet issues a GET that hangs until ctx is cancelled, reporting the
// request's error on the returned channel.
func startHungGet(ctx context.Context, rt http.RoundTripper, url string) <-chan error {
	done := make(chan error, 1)
	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			done <- err
			return
		}
		resp, err := rt.RoundTrip(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		done <- err
	}()
	return done
}

func entryTime(c *har.Collector) func() float64 {
	return func() float64 {
		entries := c.Entries()
		if len(entries) == 0 {
			return -1
		}
		return entries[len(entries)-1].Time
	}
}

var pendingEntry = And(
	HaveField("Pending", BeTrue()),
	HaveField("Error", BeEmpty()),
)

// completedAs matches the completed form of a pending snapshot: same request
// and _id, no longer pending, at least as long as the snapshot's elapsed time,
// and failed with cause.
func completedAs(pending har.Entry, cause error) OmegaMatcher {
	return And(
		HaveField("ID", pending.ID),
		HaveField("Request.URL", pending.Request.URL),
		HaveField("Pending", BeFalse()),
		HaveField("Error", ContainSubstring(cause.Error())),
		HaveField("Time", BeNumerically(">=", pending.Time)),
	)
}

var _ = Describe("Collector in-flight tracking", func() {
	It("reports hung requests as pending entries whose elapsed time grows, then completes them under the same id with the cancellation error", func() {
		srv := hangingServer()
		collector := har.NewCollector(fullCaptureConfig())
		client := commonshttp.NewClient().HARCollector(collector)
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		firstURL, secondURL := srv.URL+hangPath+"/first", srv.URL+hangPath+"/second"

		done := make(chan error, 2)
		hungGet := func(url string) {
			_, err := client.R(ctx).Get(url)
			done <- err
		}
		go hungGet(firstURL)
		Eventually(collector.Entries).Should(ConsistOf(And(pendingEntry, HaveField("Request.URL", firstURL))))
		go hungGet(secondURL)
		Eventually(collector.Entries).Should(HaveLen(2))

		pending := collector.Entries()
		Expect(pending).To(HaveExactElements(
			And(pendingEntry, HaveField("Request.URL", firstURL)),
			And(pendingEntry, HaveField("Request.URL", secondURL)),
		))
		Expect(pending[0].ID).ToNot(BeEmpty())
		Expect(pending[1].ID).ToNot(BeEmpty())
		Expect(pending[0].ID).ToNot(Equal(pending[1].ID))
		Expect(pending[1].Timings.Wait).To(Equal(pending[1].Time))
		Expect(pending[1].StartedDateTime).To(MatchRegexp(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z$`))
		Eventually(entryTime(collector)).Should(BeNumerically(">", pending[1].Time))

		raw, err := json.Marshal(pending[0])
		Expect(err).ToNot(HaveOccurred())
		Expect(string(raw)).To(ContainSubstring(`"_pending":true`))
		Expect(string(raw)).To(ContainSubstring(`"_id":"` + pending[0].ID + `"`))

		cancel()
		Eventually(done).Should(Receive(MatchError(context.Canceled)))
		Eventually(done).Should(Receive(MatchError(context.Canceled)))

		completed := collector.Entries()
		Expect(completed).To(ConsistOf(
			completedAs(pending[0], context.Canceled),
			completedAs(pending[1], context.Canceled),
		))
		for _, entry := range completed {
			raw, err := json.Marshal(entry)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(raw)).ToNot(ContainSubstring(`"_pending"`))
		}
	})

	It("keeps pending entries out of MaxEntries and forwards to the handler only on completion", func() {
		srv := hangingServer()
		var (
			mu        sync.Mutex
			forwarded []har.Entry
		)
		cfg := fullCaptureConfig()
		cfg.MaxEntries = 1
		collector := har.NewCollectorWithHandler(cfg, func(e *har.Entry) {
			mu.Lock()
			defer mu.Unlock()
			forwarded = append(forwarded, *e)
		})
		collector.Add(&har.Entry{Request: har.Request{URL: "https://example.test/completed"}})
		forwardedEntries := func() []har.Entry {
			mu.Lock()
			defer mu.Unlock()
			return append([]har.Entry(nil), forwarded...)
		}

		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		done := startHungGet(ctx, collector.Middleware()(http.DefaultTransport), srv.URL+hangPath)

		Eventually(collector.Entries).Should(HaveLen(2))
		entries := collector.Entries()
		Expect(entries[0].Request.URL).To(Equal("https://example.test/completed"))
		Expect(entries[1]).To(pendingEntry)
		Expect(collector.DroppedEntries()).To(BeZero())
		Consistently(forwardedEntries, 50*time.Millisecond).Should(HaveLen(1))

		cancel()
		Eventually(done).Should(Receive(MatchError(context.Canceled)))
		Expect(collector.Entries()).To(ConsistOf(HaveField("Request.URL", "https://example.test/completed")))
		Expect(collector.DroppedEntries()).To(Equal(1))
		Expect(forwardedEntries()).To(HaveLen(2))
		Expect(forwardedEntries()[1]).To(And(
			HaveField("Pending", BeFalse()),
			HaveField("Error", ContainSubstring(context.Canceled.Error())),
		))
	})

	It("tracks requests captured by the collector's metadata middleware as pending", func() {
		srv := hangingServer()
		collector := har.NewCollector(har.DefaultConfig())
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)

		done := startHungGet(ctx, collector.MetadataMiddleware()(http.DefaultTransport), srv.URL+hangPath)

		Eventually(collector.Entries).Should(ConsistOf(And(pendingEntry, HaveField("Request.BodySize", int64(-1)))))
		cancel()
		Eventually(done).Should(Receive(MatchError(context.Canceled)))
		Expect(collector.Entries()).To(ConsistOf(And(
			HaveField("Pending", BeFalse()),
			HaveField("Error", ContainSubstring(context.Canceled.Error())),
		)))
	})

	DescribeTable("records the transport error on handler-only middleware",
		func(newMiddleware func(har.HARConfig, func(*har.Entry)) middlewares.Middleware) {
			srv := hangingServer()
			entries := make(chan har.Entry, 1)
			rt := newMiddleware(har.DefaultConfig(), func(e *har.Entry) { entries <- *e })(http.DefaultTransport)
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)

			done := startHungGet(ctx, rt, srv.URL+hangPath)
			Consistently(entries, 50*time.Millisecond).ShouldNot(Receive())
			cancel()
			Eventually(done).Should(Receive(MatchError(context.Canceled)))
			Eventually(entries).Should(Receive(And(
				HaveField("Pending", BeFalse()),
				HaveField("Error", ContainSubstring(context.Canceled.Error())),
			)))
		},
		Entry("full middleware", har.NewMiddleware),
		Entry("metadata middleware", har.NewMetadataMiddleware),
	)
})
