package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"

	"github.com/flanksource/commons/har"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HAR capture", func() {
	It("captures every redirect request once with its matching response", func() {
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, request *stdhttp.Request) {
			switch request.URL.Path {
			case "/start":
				stdhttp.Redirect(w, request, "/middle", stdhttp.StatusFound)
			case "/middle":
				stdhttp.Redirect(w, request, "/end", stdhttp.StatusFound)
			case "/end":
				w.WriteHeader(stdhttp.StatusOK)
			}
		}))
		DeferCleanup(server.Close)

		collector := har.NewCollector(har.DefaultConfig())
		response, err := NewClient().
			HARCollector(collector).
			RedirectPolicy(5).
			R(context.Background()).
			Get(server.URL + "/start")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Body.Close()).To(Succeed())

		type capturedHop struct {
			URL    string
			Status int
		}
		entries := collector.Entries()
		hops := make([]capturedHop, len(entries))
		for i, entry := range entries {
			hops[i] = capturedHop{URL: entry.Request.URL, Status: entry.Response.Status}
		}
		Expect(hops).To(Equal([]capturedHop{
			{URL: server.URL + "/start", Status: stdhttp.StatusFound},
			{URL: server.URL + "/middle", Status: stdhttp.StatusFound},
			{URL: server.URL + "/end", Status: stdhttp.StatusOK},
		}))
	})

	It("captures requests made through the legacy NTLM transport", func() {
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			w.WriteHeader(stdhttp.StatusOK)
		}))
		DeferCleanup(server.Close)

		collector := har.NewCollector(har.DefaultConfig())
		response, err := NewClient().
			Auth("alice", "secret").
			NTLM(true).
			HARCollector(collector).
			R(context.Background()).
			Get(server.URL)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Body.Close()).To(Succeed())

		entries := collector.Entries()
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Request.URL).To(Equal(server.URL))
		Expect(entries[0].Response.Status).To(Equal(stdhttp.StatusOK))
	})
})
