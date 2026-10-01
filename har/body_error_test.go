package har_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/flanksource/commons/har"
	commonshttp "github.com/flanksource/commons/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const partialBody = `{"partial":`

// stallingBodyServer sends the headers and the first bytes of a JSON body,
// flushes them, then stalls until the client's context gives up.
func stallingBodyServer() *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, partialBody)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	DeferCleanup(srv.Close)
	return srv
}

func captureConfig(maxBodySize int64) *har.HARConfig {
	cfg := har.DefaultConfig()
	cfg.MaxBodySize = maxBodySize
	return &cfg
}

var _ = Describe("Response body read errors", func() {
	DescribeTable("surface to the caller unchanged and are recorded on the entry",
		func(cfg *har.HARConfig) {
			srv := stallingBodyServer()
			client := commonshttp.NewClient()
			var collector *har.Collector
			if cfg != nil {
				collector = har.NewCollector(*cfg)
				client = client.HARCollector(collector)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			DeferCleanup(cancel)

			resp, err := client.R(ctx).Get(srv.URL)
			Expect(err).ToNot(HaveOccurred())
			body, readErr := io.ReadAll(resp.Body)
			Expect(resp.Body.Close()).To(Succeed())

			Expect(string(body)).To(Equal(partialBody))
			Expect(errors.Is(readErr, context.DeadlineExceeded)).To(BeTrue(), "read error %v must wrap context.DeadlineExceeded", readErr)
			if collector == nil {
				return
			}
			Expect(collector.Entries()).To(ConsistOf(And(
				HaveField("Pending", BeFalse()),
				HaveField("Error", Equal(readErr.Error())),
				HaveField("Response.Content.Text", partialBody),
				HaveField("Response.Status", http.StatusOK),
			)))
		},
		Entry("without HAR capture", nil),
		Entry("with full capture", captureConfig(0)),
		Entry("with bounded capture", captureConfig(1024)),
	)
})
