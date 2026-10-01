package har_test

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/http/middlewares"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("durable HAR lifecycle", func() {
	It("emits a request before the transport runs and a completion with its stable ID without retaining bodies", func() {
		var events []har.Entry
		collector := har.NewCollectorWithLifecycle(fullCaptureConfig(), func(entry *har.Entry) error {
			events = append(events, *entry)
			return nil
		}, func(entry *har.Entry) error {
			events = append(events, *entry)
			return nil
		})
		roundTrip := collector.Middleware()(middlewares.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			Expect(events).To(HaveLen(1))
			Expect(events[0].Pending).To(BeTrue())
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		}))
		request, err := http.NewRequest(http.MethodGet, "https://example.test/trace", nil)
		Expect(err).NotTo(HaveOccurred())
		response, err := roundTrip.RoundTrip(request)
		Expect(err).NotTo(HaveOccurred())
		Expect(response.Body.Close()).To(Succeed())
		Expect(events).To(HaveLen(2))
		Expect(events[0].ID).NotTo(BeEmpty())
		Expect(events[1].ID).To(Equal(events[0].ID))
		Expect(events[1].Pending).To(BeFalse())
		Expect(events[1].Response.Content.Text).To(Equal(`{"ok":true}`))
		Expect(collector.Entries()).To(BeEmpty())
	})

	It("retains lifecycle write errors for the owner without changing the HTTP response", func() {
		writeError := errors.New("capture store unavailable")
		collector := har.NewCollectorWithLifecycle(fullCaptureConfig(), func(*har.Entry) error { return writeError }, func(*har.Entry) error { return nil })
		roundTrip := collector.Middleware()(middlewares.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
		}))
		request, err := http.NewRequest(http.MethodGet, "https://example.test/trace", nil)
		Expect(err).NotTo(HaveOccurred())
		response, err := roundTrip.RoundTrip(request)
		Expect(err).NotTo(HaveOccurred())
		Expect(response.StatusCode).To(Equal(http.StatusNoContent))
		Expect(collector.CaptureError()).To(MatchError(writeError))
	})
})
