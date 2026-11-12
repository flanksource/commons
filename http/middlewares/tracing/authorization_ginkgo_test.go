package tracing

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	commonscontext "github.com/flanksource/commons/context"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("HTTP logging of echoed credentials", func(structured, responseBody bool, authorization string) {
	const body = `{"message":"denied example-private-token"}`
	var output bytes.Buffer
	log := logger.NewWithWriter(&output)
	ctx := commonscontext.NewContext(context.Background(), commonscontext.WithLogger(log))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test", nil)
	Expect(err).NotTo(HaveOccurred())
	request.Header.Set("Authorization", authorization)
	transport := middlewares.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewBufferString(body))}, nil
	})
	render := prettyLogger
	if structured {
		render = jsonLogger
	}
	response, err := render(TraceConfig{AccessLog: true, Response: responseBody}, nil, transport, request)
	Expect(err).NotTo(HaveOccurred())
	Expect(output.String()).To(SatisfyAll(ContainSubstring("denied"), Not(ContainSubstring("example-private-token"))))
	data, err := io.ReadAll(response.Body)
	Expect(err).NotTo(HaveOccurred())
	Expect(response.Body.Close()).To(Succeed())
	Expect(string(data)).To(Equal(body))
},
	Entry("pretty errors", false, false, "Bearer example-private-token"),
	Entry("pretty response bodies", false, true, "Bearer example-private-token"),
	Entry("JSON errors", true, false, "Bearer example-private-token"),
	Entry("JSON response bodies", true, true, "Bearer example-private-token"),
	Entry("basic errors", false, false, "Basic YXBpOmV4YW1wbGUtcHJpdmF0ZS10b2tlbg=="),
)

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

var _ = It("redacts credentials echoed by response body read errors in JSON logs", func() {
	var output bytes.Buffer
	log := logger.NewWithWriter(&output)
	ctx := commonscontext.NewContext(context.Background(), commonscontext.WithLogger(log))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test", nil)
	Expect(err).NotTo(HaveOccurred())
	request.Header.Set("Authorization", "Bearer example-private-token")
	transport := middlewares.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
		readErr := errors.New("stream reset while sending example-private-token")
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(failingReader{err: readErr})}, nil
	})
	_, err = jsonLogger(TraceConfig{AccessLog: true, Response: true}, nil, transport, request)
	Expect(err).NotTo(HaveOccurred())
	Expect(output.String()).To(SatisfyAll(ContainSubstring("stream reset"), Not(ContainSubstring("example-private-token"))))
})
