package middlewares

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync"
	"sync/atomic"

	rpchttp "github.com/flanksource/clicky/rpc/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server timing", func() {
	It("counts exchanges and connection reuse without consuming the body", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "payload") }))
		DeferCleanup(server.Close)
		transport := http.DefaultTransport.(*http.Transport).Clone()
		DeferCleanup(transport.CloseIdleConnections)
		client := &http.Client{Transport: ServerTiming(transport)}
		ctx, timings := rpchttp.WithTimings(context.Background())
		var traced atomic.Int64
		ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{GotConn: func(httptrace.GotConnInfo) { traced.Add(1) }})
		for range 2 {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			Expect(err).NotTo(HaveOccurred())
			response, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			body, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("payload"))
			Expect(response.Body.Close()).To(Succeed())
		}
		requests, _ := timings.Counter("http", "requests")
		Expect(requests).To(Equal(int64(2)))
		reused, _ := timings.Counter("http_conn", "reused")
		Expect(reused).To(Equal(int64(1)))
		bytes, _ := timings.Counter("http_body", "bytes")
		Expect(bytes).To(Equal(int64(14)))
		bodies, _ := timings.Counter("http_body", "responses")
		Expect(bodies).To(Equal(int64(2)))
		Expect(traced.Load()).To(Equal(int64(2)))
	})

	It("records transport errors without changing them", func() {
		failure := errors.New("dial failed")
		ctx, timings := rpchttp.WithTimings(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", nil)
		Expect(err).NotTo(HaveOccurred())
		_, err = ServerTiming(RoundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, failure })).RoundTrip(req)
		Expect(err).To(MatchError(failure))
		count, _ := timings.Counter("http", "errors")
		Expect(count).To(Equal(int64(1)))
	})

	It("keeps concurrent requests isolated and finalizes an unread body once", func() {
		transport := ServerTiming(RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unread")), Request: req}, nil
		}))
		var group sync.WaitGroup
		for range 4 {
			group.Go(func() {
				defer GinkgoRecover()
				ctx, timings := rpchttp.WithTimings(context.Background())
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := transport.RoundTrip(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Body.Close()).To(Succeed())
				Expect(resp.Body.Close()).To(Succeed())
				requests, _ := timings.Counter("http", "requests")
				Expect(requests).To(Equal(int64(1)))
				bodies, _ := timings.Counter("http_body", "responses")
				Expect(bodies).To(Equal(int64(1)))
			})
		}
		group.Wait()
	})

	It("does not change the response body or context when no collector is installed", func() {
		ctx := context.Background()
		body := io.NopCloser(strings.NewReader("stream"))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", nil)
		Expect(err).NotTo(HaveOccurred())
		response, err := ServerTiming(RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			Expect(r).To(BeIdenticalTo(req))
			return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
		})).RoundTrip(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(response.Body).To(BeIdenticalTo(body))
		Expect(WithClientTimings(ctx)).To(BeIdenticalTo(ctx))
		Expect(response.Body.Close()).To(Succeed())
	})

	It("preserves body read failures and records the completion once", func() {
		failure := io.ErrUnexpectedEOF
		ctx, timings := rpchttp.WithTimings(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", nil)
		Expect(err).NotTo(HaveOccurred())
		response, err := ServerTiming(RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(failedTimingReader{err: failure})}, nil
		})).RoundTrip(req)
		Expect(err).NotTo(HaveOccurred())
		_, err = io.ReadAll(response.Body)
		Expect(err).To(MatchError(failure))
		Expect(response.Body.Close()).To(Succeed())
		count, _ := timings.Counter("http_body", "errors")
		Expect(count).To(Equal(int64(1)))
		completions, _ := timings.Counter("http_body", "responses")
		Expect(completions).To(Equal(int64(1)))
	})
})

type failedTimingReader struct{ err error }

func (r failedTimingReader) Read([]byte) (int, error) { return 0, r.err }
