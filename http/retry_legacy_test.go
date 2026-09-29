package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const legacyRetryWait = 5 * time.Second

// attemptCountingServer counts every attempt it receives and runs respond for
// it with the 1-based attempt number.
func attemptCountingServer(respond func(w stdhttp.ResponseWriter, attempt int32)) (*httptest.Server, *atomic.Int32) {
	var attempts atomic.Int32
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		respond(w, attempts.Add(1))
	}))
	DeferCleanup(srv.Close)
	return srv, &attempts
}

// dropConnection aborts the attempt with a transport error on the client side.
func dropConnection(w stdhttp.ResponseWriter, _ int32) {
	conn, _, err := w.(stdhttp.Hijacker).Hijack()
	Expect(err).ToNot(HaveOccurred())
	Expect(conn.Close()).To(Succeed())
}

var _ = Describe("Legacy Retry", func() {
	It("returns promptly without a backoff sleep once the context has expired", func() {
		srv, attempts := attemptCountingServer(dropConnection)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		DeferCleanup(cancel)

		started := time.Now()
		_, err := NewClient().Retry(3, legacyRetryWait, 1).R(ctx).Get(srv.URL)

		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(time.Since(started)).To(BeNumerically("<", legacyRetryWait))
		Expect(attempts.Load()).To(BeZero())
	})

	It("abandons the backoff wait when the context expires during it", func() {
		srv, attempts := attemptCountingServer(dropConnection)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		started := time.Now()
		_, err := NewClient().Retry(3, legacyRetryWait, 1).R(ctx).Get(srv.URL)

		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "error %v must wrap context.DeadlineExceeded", err)
		Expect(time.Since(started)).To(BeNumerically("<", legacyRetryWait))
		Expect(attempts.Load()).To(Equal(int32(1)))
	})

	DescribeTable("replays only idempotent methods after a transport error",
		func(method string, wantAttempts int32, wantErr bool) {
			srv, attempts := attemptCountingServer(func(w stdhttp.ResponseWriter, attempt int32) {
				if attempt == 1 {
					time.Sleep(200 * time.Millisecond)
				}
				w.WriteHeader(stdhttp.StatusOK)
			})

			resp, err := NewClient().
				BaseURL(srv.URL).
				Timeout(50*time.Millisecond).
				Retry(3, time.Millisecond, 1).
				R(context.Background()).
				Do(method, "/")

			Expect(attempts.Load()).To(Equal(wantAttempts))
			if wantErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(stdhttp.StatusOK))
		},
		Entry("GET is retried", stdhttp.MethodGet, int32(2), false),
		Entry("PUT is retried", stdhttp.MethodPut, int32(2), false),
		Entry("DELETE is retried", stdhttp.MethodDelete, int32(2), false),
		Entry("POST is not replayed", stdhttp.MethodPost, int32(1), true),
		Entry("PATCH is not replayed", stdhttp.MethodPatch, int32(1), true),
	)
})
