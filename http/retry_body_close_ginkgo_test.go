package http

import (
	"context"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RetryStrategy discarded responses", func() {
	It("closes every retried response body so the connection is reused", func() {
		var attempts, newConnections int32
		server := httptest.NewUnstartedServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			if atomic.AddInt32(&attempts, 1) <= 2 {
				w.WriteHeader(stdhttp.StatusServiceUnavailable)
				_, _ = w.Write([]byte("temporarily unavailable"))
				return
			}
			_, _ = w.Write([]byte("ok"))
		}))
		server.Config.ConnState = func(_ net.Conn, state stdhttp.ConnState) {
			if state == stdhttp.StateNew {
				atomic.AddInt32(&newConnections, 1)
			}
		}
		server.Start()
		defer server.Close()

		strategy := func(resp *Response, _ error, _ int) (bool, time.Duration) {
			return resp.StatusCode == stdhttp.StatusServiceUnavailable, 0
		}
		response, err := NewClient().RetryStrategy(strategy).R(context.Background()).Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(response.StatusCode).To(Equal(stdhttp.StatusOK))
		body, err := response.AsString()
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(Equal("ok"))
		Expect(atomic.LoadInt32(&attempts)).To(BeEquivalentTo(3))
		Expect(atomic.LoadInt32(&newConnections)).To(BeEquivalentTo(1))
	})
})
