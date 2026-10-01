package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WithContext metadata HAR", func() {
	It("shows an in-flight request as a pending collector entry until it completes", func() {
		srv := httptest.NewServer(stdhttp.HandlerFunc(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
			<-r.Context().Done()
		}))
		DeferCleanup(srv.Close)

		collector := har.NewCollector(har.DefaultConfig())
		client := NewClient().WithContext(&fakeContext{
			log:      logger.New("har-inflight"),
			harColl:  collector,
			harPath:  "/dev/null",
			harLevel: HARMetadata,
		}, "inflight")
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)

		done := make(chan error, 1)
		go func() {
			_, err := client.R(ctx).Get(srv.URL)
			done <- err
		}()

		Eventually(collector.Entries).Should(ConsistOf(And(
			HaveField("Pending", BeTrue()),
			HaveField("Request.URL", srv.URL),
		)))
		cancel()
		Eventually(done).Should(Receive(MatchError(context.Canceled)))
		Expect(collector.Entries()).To(ConsistOf(And(
			HaveField("Pending", BeFalse()),
			HaveField("Error", ContainSubstring(context.Canceled.Error())),
		)))
	})
})
