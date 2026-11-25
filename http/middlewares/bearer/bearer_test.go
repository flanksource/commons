package bearer_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/http/middlewares/bearer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
)

func TestBearer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bearer Middleware Suite")
}

type fakeSource struct {
	calls atomic.Int32
	token func(call int32) (*oauth2.Token, error)
}

func (f *fakeSource) Token() (*oauth2.Token, error) { return f.token(f.calls.Add(1)) }

type recordingTransport struct {
	requests []*http.Request
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.requests = append(r.requests, req)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Request: req}, nil
}

func do(mw middlewares.Middleware, base http.RoundTripper, header http.Header) (*http.Response, error) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	for k, v := range header {
		req.Header[k] = v
	}
	return mw(base).RoundTrip(req)
}

var _ = Describe("bearer.New", func() {
	It("sets the Authorization header and does not mutate the caller's request", func() {
		src := &fakeSource{token: func(int32) (*oauth2.Token, error) {
			return &oauth2.Token{AccessToken: "tok-1", Expiry: time.Now().Add(time.Hour)}, nil
		}}
		base := &recordingTransport{}
		req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)

		_, err := bearer.New(src)(base).RoundTrip(req)

		Expect(err).NotTo(HaveOccurred())
		Expect(base.requests[0].Header.Get("Authorization")).To(Equal("Bearer tok-1"))
		Expect(req.Header.Get("Authorization")).To(BeEmpty())
	})

	It("caches the token until it nears expiry and refreshes after", func() {
		src := &fakeSource{token: func(call int32) (*oauth2.Token, error) {
			expiry := time.Now().Add(time.Hour)
			if call == 1 {
				expiry = time.Now().Add(time.Second) // inside the refresh window
			}
			return &oauth2.Token{AccessToken: "tok", Expiry: expiry}, nil
		}}
		mw := bearer.New(src)
		base := &recordingTransport{}

		for range 3 {
			_, err := do(mw, base, nil)
			Expect(err).NotTo(HaveOccurred())
		}

		Expect(src.calls.Load()).To(Equal(int32(2)), "first token expired early, second one is reused")
	})

	It("fails without sending when the source errors", func() {
		src := &fakeSource{token: func(int32) (*oauth2.Token, error) { return nil, errors.New("boom") }}
		base := &recordingTransport{}

		_, err := do(bearer.New(src), base, nil)

		Expect(err).To(MatchError(ContainSubstring("boom")))
		Expect(base.requests).To(BeEmpty())
	})

	It("fails without sending when the token is empty", func() {
		src := &fakeSource{token: func(int32) (*oauth2.Token, error) {
			return &oauth2.Token{Expiry: time.Now().Add(time.Hour)}, nil
		}}
		base := &recordingTransport{}

		_, err := do(bearer.New(src), base, nil)

		Expect(err).To(MatchError(ContainSubstring("empty access token")))
		Expect(base.requests).To(BeEmpty())
	})

	It("rejects a request that already has an Authorization header", func() {
		src := &fakeSource{token: func(int32) (*oauth2.Token, error) {
			return &oauth2.Token{AccessToken: "tok", Expiry: time.Now().Add(time.Hour)}, nil
		}}
		base := &recordingTransport{}

		_, err := do(bearer.New(src), base, http.Header{"Authorization": {"Basic abc"}})

		Expect(err).To(MatchError(ContainSubstring("already has an Authorization header")))
		Expect(base.requests).To(BeEmpty())
		Expect(src.calls.Load()).To(BeZero())
	})

	It("fails every request when the source is nil", func() {
		base := &recordingTransport{}

		_, err := do(bearer.New(nil), base, nil)

		Expect(err).To(MatchError(ContainSubstring("nil TokenSource")))
		Expect(base.requests).To(BeEmpty())
	})
})
