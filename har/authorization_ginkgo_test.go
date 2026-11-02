package har

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.DescribeTable("HAR capture of echoed credentials", func(sensitive bool) {
	const body = `{"message":"denied example-private-token"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, body)
	}))
	ginkgo.DeferCleanup(server.Close)
	config := DefaultConfig()
	config.CaptureSensitive = sensitive
	collector := NewCollector(config)
	client := &http.Client{Transport: collector.Middleware()(http.DefaultTransport)}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	request.Header.Set("Authorization", "Bearer example-private-token")
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	Expect(err).NotTo(HaveOccurred())
	data, err := io.ReadAll(response.Body)
	Expect(err).NotTo(HaveOccurred())
	Expect(response.Body.Close()).To(Succeed())
	Expect(string(data)).To(Equal(body))
	Expect(collector.Entries()).To(HaveLen(1))
	entry := collector.Entries()[0]
	Expect(strings.Contains(entry.Request.PostData.Text, "example-private-token")).To(Equal(sensitive))
	Expect(strings.Contains(entry.Response.Content.Text, "example-private-token")).To(Equal(sensitive))
}, ginkgo.Entry("default redaction", false), ginkgo.Entry("explicit sensitive capture", true))
