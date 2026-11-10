package http

import (
	"context"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/flanksource/commons/http/middlewares"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type countingWrapper struct {
	next  stdhttp.RoundTripper
	calls *atomic.Int32
}

func (w *countingWrapper) RoundTrip(request *stdhttp.Request) (*stdhttp.Response, error) {
	w.calls.Add(1)
	return w.next.RoundTrip(request)
}

func (w *countingWrapper) Unwrap() stdhttp.RoundTripper { return w.next }

func (w *countingWrapper) Rewrap(next stdhttp.RoundTripper) stdhttp.RoundTripper {
	return &countingWrapper{next: next, calls: w.calls}
}

var _ TransportWrapper = (*countingWrapper)(nil)

var _ = Describe("Wrapped transports", func() {
	const proxiedBody = "via-proxy"
	var original stdhttp.RoundTripper

	BeforeEach(func() { original = stdhttp.DefaultTransport })
	AfterEach(func() { stdhttp.DefaultTransport = original })

	It("configures the inner transport of a wrapped DefaultTransport and keeps the wrapper outermost", func() {
		proxy := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			_, _ = io.WriteString(w, proxiedBody)
		}))
		defer proxy.Close()

		var calls atomic.Int32
		raw := original.(*stdhttp.Transport).Clone()
		stdhttp.DefaultTransport = &countingWrapper{next: raw, calls: &calls}

		client := NewClient().InsecureSkipVerify(true).DisableKeepAlive(true).Proxy(proxy.URL)
		response, err := client.R(context.Background()).Get("http://upstream.example/data")
		Expect(err).NotTo(HaveOccurred())
		Expect(response.AsString()).To(Equal(proxiedBody))
		Expect(calls.Load()).To(Equal(int32(1)))

		wrapper, ok := client.httpClient.Transport.(*countingWrapper)
		Expect(ok).To(BeTrue(), "client transport is %T", client.httpClient.Transport)
		inner, ok := wrapper.next.(*stdhttp.Transport)
		Expect(ok).To(BeTrue(), "inner transport is %T", wrapper.next)
		Expect(inner).NotTo(BeIdenticalTo(raw))
		Expect(inner.TLSClientConfig.InsecureSkipVerify).To(BeTrue())
		Expect(inner.DisableKeepAlives).To(BeTrue())
		selected, err := inner.Proxy(httptest.NewRequest(stdhttp.MethodGet, "http://upstream.example/data", nil))
		Expect(err).NotTo(HaveOccurred())
		Expect(selected).To(Equal(mustParseURL(proxy.URL)))
		Expect(raw.TLSClientConfig == nil || !raw.TLSClientConfig.InsecureSkipVerify).To(BeTrue())
	})

	It("fails requests when the wrapped chain does not end in an *http.Transport", func() {
		var calls atomic.Int32
		opaque := middlewares.RoundTripperFunc(func(*stdhttp.Request) (*stdhttp.Response, error) {
			Fail("opaque transport must not be reached")
			return nil, nil
		})
		stdhttp.DefaultTransport = &countingWrapper{next: opaque, calls: &calls}

		_, err := NewClient().InsecureSkipVerify(true).R(context.Background()).Get("http://upstream.example/data")
		Expect(err).To(MatchError(And(
			ContainSubstring("middlewares.RoundTripperFunc"),
			ContainSubstring("*http.Transport"),
		)))
		Expect(calls.Load()).To(BeZero())
	})

	It("applies a connect timeout through a wrapped DefaultTransport", func() {
		var calls atomic.Int32
		dialed := make(chan struct{}, 1)
		raw := &stdhttp.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialed <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		}}
		stdhttp.DefaultTransport = &countingWrapper{next: raw, calls: &calls}

		client, err := NewClient().ConnectTimeout(25 * time.Millisecond)
		Expect(err).NotTo(HaveOccurred())
		_, err = client.httpClient.Get("http://upstream.example/data")
		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue(), "error: %v", err)
		Expect(dialed).To(Receive())
		Expect(calls.Load()).To(Equal(int32(1)))
		Expect(client.httpClient.Transport).To(BeAssignableToTypeOf(&countingWrapper{}))
	})

	It("pools the inner transport of a wrapped DefaultTransport", func() {
		var calls atomic.Int32
		stdhttp.DefaultTransport = &countingWrapper{next: original.(*stdhttp.Transport).Clone(), calls: &calls}
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			_, _ = io.WriteString(w, proxiedBody)
		}))
		defer server.Close()

		pool, err := NewTransportPool(TransportPoolOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer func() { Expect(pool.Close()).To(Succeed()) }()
		client, err := pool.Client(TransportOptions{Endpoint: server.URL, Proxy: server.URL})
		Expect(err).NotTo(HaveOccurred())
		again, err := pool.Client(TransportOptions{Endpoint: server.URL, Proxy: server.URL})
		Expect(err).NotTo(HaveOccurred())
		Expect(again.httpClient.Transport).To(BeIdenticalTo(client.httpClient.Transport))

		response, err := client.R(context.Background()).Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(response.AsString()).To(Equal(proxiedBody))
		Expect(calls.Load()).To(Equal(int32(1)))
	})
})

var _ = Describe("Transport configuration after Use", func() {
	opaque := func(next stdhttp.RoundTripper) stdhttp.RoundTripper {
		return middlewares.RoundTripperFunc(next.RoundTrip)
	}
	baseOf := func(client *Client) *stdhttp.Transport {
		transport, ok := client.baseTransport.(*stdhttp.Transport)
		Expect(ok).To(BeTrue(), "base transport is %T", client.baseTransport)
		return transport
	}

	It("configures the base transport beneath opaque middleware", func() {
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			_, _ = io.WriteString(w, "ok")
		}))
		defer server.Close()

		client := NewClient().Use(opaque).DisableKeepAlive(true).Proxy(server.URL)
		_, err := client.TLSConfig(TLSConfig{InsecureSkipVerify: true})
		Expect(err).NotTo(HaveOccurred())

		response, err := client.R(context.Background()).Get("http://upstream.example/data")
		Expect(err).NotTo(HaveOccurred())
		Expect(response.AsString()).To(Equal("ok"))
		Expect(baseOf(client).DisableKeepAlives).To(BeTrue())
		Expect(baseOf(client).TLSClientConfig.InsecureSkipVerify).To(BeTrue())
	})

	It("keeps certificate verification re-enabled across a later Use", func() {
		client := NewClient().InsecureSkipVerify(true).Use(opaque).InsecureSkipVerify(false).Use(opaque)
		Expect(baseOf(client).TLSClientConfig.InsecureSkipVerify).To(BeFalse())
		Expect(client.transportMiddlewares).To(HaveLen(2))
	})
})

func mustParseURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	Expect(err).NotTo(HaveOccurred())
	return parsed
}
