package http

import (
	"context"
	"io"
	stdhttp "net/http"
	"strings"

	"github.com/flanksource/commons/http/middlewares"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// capturingClient records the URL host of each outgoing request instead of
// sending it, so host rewriting can be asserted without DNS or a listener.
func capturingClient(hosts *[]string) *Client {
	return NewClient().Use(func(stdhttp.RoundTripper) stdhttp.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
			*hosts = append(*hosts, req.URL.Host)
			return &stdhttp.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: stdhttp.Header{}, Request: req}, nil
		})
	})
}

var _ = Describe("request URL host", func() {
	DescribeTable("keeps the host exactly as given, without a dangling port separator",
		func(rawURL, expectedHost string) {
			var hosts []string
			_, err := capturingClient(&hosts).R(context.Background()).Get(rawURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(hosts).To(Equal([]string{expectedHost}))
		},
		Entry("https without a port", "https://sts.us-east-1.amazonaws.com/", "sts.us-east-1.amazonaws.com"),
		Entry("http without a port", "http://example.com/path", "example.com"),
		Entry("explicit port", "http://example.com:8080/path", "example.com:8080"),
		Entry("IPv6 literal without a port", "http://[::1]/path", "[::1]"),
		Entry("IPv6 literal with a port", "http://[::1]:9000/path", "[::1]:9000"),
	)
})
