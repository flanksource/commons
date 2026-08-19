package http

import (
	"context"
	"errors"
	"net"
	stdhttp "net/http"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConnectTimeout(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP Connect Timeout Suite")
}

var _ = Describe("Client connect timeout", func() {
	It("rejects non-positive durations", func() {
		for _, timeout := range []time.Duration{0, -time.Nanosecond} {
			_, err := NewClient().ConnectTimeout(timeout)
			Expect(err).To(MatchError(ContainSubstring("connect timeout must be greater than zero")), timeout)
		}
	})

	It("bounds the configured dialer and preserves client settings", func() {
		var dialAttempts atomic.Int32
		originalTransport := &stdhttp.Transport{
			DisableKeepAlives: true,
			MaxIdleConns:      17,
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialAttempts.Add(1)
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}
		client := NewClient().Timeout(time.Second)
		client.httpClient.Transport = originalTransport

		configured, err := client.ConnectTimeout(25 * time.Millisecond)
		Expect(err).ToNot(HaveOccurred())
		Expect(configured).To(BeIdenticalTo(client))
		transport := client.httpClient.Transport.(*stdhttp.Transport)
		Expect(transport).ToNot(BeIdenticalTo(originalTransport))
		Expect(transport.DisableKeepAlives).To(BeTrue())
		Expect(transport.MaxIdleConns).To(Equal(17))
		Expect(client.httpClient.Timeout).To(Equal(time.Second))

		started := time.Now()
		_, err = client.httpClient.Get("http://example.test")
		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
		Expect(time.Since(started)).To(BeNumerically(">=", 25*time.Millisecond))
		Expect(time.Since(started)).To(BeNumerically("<", time.Second))
		Expect(dialAttempts.Load()).To(Equal(int32(1)))
	})

	It("honors an earlier request deadline", func() {
		client := NewClient()
		client.httpClient.Transport = &stdhttp.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}}
		_, err := client.ConnectTimeout(time.Second)
		Expect(err).ToNot(HaveOccurred())

		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()
		request, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodGet, "http://example.test", nil)
		Expect(err).ToNot(HaveOccurred())
		started := time.Now()
		_, err = client.httpClient.Do(request)
		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
		Expect(time.Since(started)).To(BeNumerically("<", time.Second))
	})

	It("replaces an earlier connection timeout", func() {
		dialCompleted := errors.New("dial completed after the first timeout")
		client := NewClient()
		client.httpClient.Transport = &stdhttp.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			timer := time.NewTimer(50 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-timer.C:
				return nil, dialCompleted
			}
		}}

		_, err := client.ConnectTimeout(10 * time.Millisecond)
		Expect(err).ToNot(HaveOccurred())
		_, err = client.ConnectTimeout(100 * time.Millisecond)
		Expect(err).ToNot(HaveOccurred())

		_, err = client.httpClient.Get("http://example.test")
		Expect(errors.Is(err, dialCompleted)).To(BeTrue())
	})
})
