package awssigv4

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/flanksource/commons/http/middlewares"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type failingBody struct{ reads int }

func (b *failingBody) Read([]byte) (int, error) { b.reads++; return 0, io.ErrUnexpectedEOF }
func (*failingBody) Close() error               { return nil }

var _ = Describe("SigV4 transport", func() {
	It("buffers a non-rewindable body and forwards the exact payload without mutating request headers", func() {
		const body = `{"message":"hello"}`
		request, err := http.NewRequest("POST", "https://s3.amazonaws.com/object", io.NopCloser(strings.NewReader(body)))
		Expect(err).NotTo(HaveOccurred())
		Expect(request.GetBody).To(BeNil())
		transport := NewAWSSigv4Transport(AWSSigv4Config{Region: "us-east-1", CredentialsProvider: NewStaticCredentialsProvider(Credentials{AccessKeyID: "example", SecretAccessKey: "example-secret", SessionToken: "example-session"})}, middlewares.RoundTripperFunc(func(signed *http.Request) (*http.Response, error) {
			data, err := io.ReadAll(signed.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(body))
			Expect(signed.Header.Get("X-Amz-Content-Sha256")).To(Equal(hashPayload([]byte(body))))
			Expect(signed.Header.Get("X-Amz-Security-Token")).To(Equal("example-session"))
			Expect(signed.Header.Get("Authorization")).To(HavePrefix("AWS4-HMAC-SHA256 "))
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		}))
		_, err = transport.RoundTrip(request)
		Expect(err).NotTo(HaveOccurred())
		Expect(request.Header).To(BeEmpty())
	})

	It("supports explicitly unsigned streaming payloads and propagates body and provider failures", func() {
		body := &failingBody{}
		request, err := http.NewRequest("POST", "https://s3.amazonaws.com/object", body)
		Expect(err).NotTo(HaveOccurred())
		config := AWSSigv4Config{Region: "us-east-1", UnsignedPayload: true, CredentialsProvider: NewStaticCredentialsProvider(Credentials{AccessKeyID: "example", SecretAccessKey: "example-secret"})}
		var calls int
		next := middlewares.RoundTripperFunc(func(signed *http.Request) (*http.Response, error) {
			calls++
			Expect(signed.Body).To(BeIdenticalTo(body))
			Expect(signed.Header.Get("X-Amz-Content-Sha256")).To(Equal("UNSIGNED-PAYLOAD"))
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		})
		_, err = NewAWSSigv4Transport(config, next).RoundTrip(request)
		Expect(err).NotTo(HaveOccurred())
		Expect(body.reads).To(BeZero())
		config.UnsignedPayload = false
		_, err = NewAWSSigv4Transport(config, next).RoundTrip(request)
		Expect(errors.Is(err, io.ErrUnexpectedEOF)).To(BeTrue())
		failure := errors.New("credentials unavailable")
		config.CredentialsProvider = CredentialsProviderFunc(func(context.Context) (Credentials, error) { return Credentials{}, failure })
		_, err = NewAWSSigv4Transport(config, next).RoundTrip(request)
		Expect(errors.Is(err, failure)).To(BeTrue())
		Expect(calls).To(Equal(1))
	})
})
