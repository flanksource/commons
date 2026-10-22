package http

import (
	"context"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	rpchttp "github.com/flanksource/clicky/rpc/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server timing shared client", func() {
	It("counts redirect hops and retry attempts through the normal fluent client", func() {
		var attempts atomic.Int64
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, req *stdhttp.Request) {
			if req.URL.Path == "/start" {
				stdhttp.Redirect(w, req, "/end", stdhttp.StatusFound)
				return
			}
			if attempts.Add(1) == 1 {
				w.WriteHeader(stdhttp.StatusServiceUnavailable)
				return
			}
			_, _ = io.WriteString(w, "complete")
		}))
		DeferCleanup(server.Close)
		ctx, timings := rpchttp.WithTimings(context.Background())
		response, err := NewClient().RedirectPolicy(5).RetryStrategy(RetryOnStatus(2, time.Millisecond, stdhttp.StatusServiceUnavailable)).R(ctx).Get(server.URL + "/start")
		Expect(err).NotTo(HaveOccurred())
		body, err := response.AsString()
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(Equal("complete"))
		requests, _ := timings.Counter("http", "requests")
		Expect(requests).To(Equal(int64(4)))
		failures, _ := timings.Counter("http", "status_errors")
		Expect(failures).To(Equal(int64(1)))
	})

	It("passes cancellation through and records it on the request", func() {
		ctx, timings := rpchttp.WithTimings(context.Background())
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		_, err := NewClient().R(ctx).Get("http://example.invalid")
		Expect(err).To(MatchError(ContainSubstring("context canceled")))
		failures, _ := timings.Counter("http", "errors")
		Expect(failures).To(Equal(int64(1)))
	})
})
