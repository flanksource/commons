package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"time"

	"github.com/flanksource/commons/har"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
)

type staticTokenSource string

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: string(s), Expiry: time.Now().Add(time.Hour)}, nil
}

var _ = Describe("Client.BearerToken", func() {
	const secretToken = "super-secret-bearer-token-value"

	var (
		seen      chan string
		serverURL string
	)

	BeforeEach(func() {
		seen = make(chan string, 4)
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			seen <- r.Header.Get("Authorization")
		}))
		DeferCleanup(server.Close)
		serverURL = server.URL
	})

	It("sends the bearer token and redacts it in the HAR capture", func() {
		collector := har.NewCollector(har.DefaultConfig())

		response, err := NewClient().
			HARCollector(collector).
			BearerToken(staticTokenSource(secretToken)).
			R(context.Background()).
			Get(serverURL)

		Expect(err).ToNot(HaveOccurred())
		Expect(response.Body.Close()).To(Succeed())
		Expect(<-seen).To(Equal("Bearer " + secretToken))

		entries := collector.Entries()
		Expect(entries).To(HaveLen(1))
		var authorization string
		for _, h := range entries[0].Request.Headers {
			if h.Name == "Authorization" {
				authorization = h.Value
			}
		}
		Expect(authorization).ToNot(BeEmpty(), "HAR sees the header because bearer wraps the HAR middleware")
		Expect(authorization).ToNot(ContainSubstring(secretToken))
	})

	It("refuses a request that already carries an Authorization header", func() {
		_, err := NewClient().
			BearerToken(staticTokenSource(secretToken)).
			R(context.Background()).
			Header("Authorization", "Basic abc").
			Get(serverURL)

		Expect(err).To(MatchError(ContainSubstring("already has an Authorization header")))
		Expect(seen).To(BeEmpty())
	})
})
