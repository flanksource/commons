package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP response body logging limits", func() {
	const (
		bodyLimit = 8
		body      = "abcdefgh-remaining"
	)

	var output bytes.Buffer
	var previousOutput io.Writer

	BeforeEach(func() {
		output.Reset()
		previousOutput = logger.GetOutput()
		logger.SetOutput(&output)
		properties.Set("http.log.response.body.length", bodyLimit)
	})

	AfterEach(func() {
		properties.Set("http.log.response.body.length", "")
		logger.SetOutput(previousOutput)
	})

	DescribeTable("retains a bounded prefix and restores the response",
		func(writeResponse func(stdhttp.ResponseWriter)) {
			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
				w.Header().Set("Content-Type", "text/plain")
				writeResponse(w)
			}))
			DeferCleanup(server.Close)

			log := logger.New("response-limit")
			log.SetLogLevel(logger.Trace2)
			response, err := NewClient().WithLogger(log).R(context.Background()).Get(server.URL)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(response.Body.Close)

			logged := output.String()
			Expect(logged).To(ContainSubstring(body[:bodyLimit]))
			Expect(logged).To(ContainSubstring("body truncated after 8 bytes"))
			Expect(logged).ToNot(ContainSubstring(body[bodyLimit:]))

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(responseBody)).To(Equal(body))
		},
		Entry("for a known content length", func(w stdhttp.ResponseWriter) {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
			_, _ = io.WriteString(w, body)
		}),
		Entry("for a chunked response", func(w stdhttp.ResponseWriter) {
			_, _ = io.WriteString(w, body[:bodyLimit])
			w.(stdhttp.Flusher).Flush()
			_, _ = io.WriteString(w, body[bodyLimit:])
		}),
	)

	It("redacts secrets inside a truncated JSON prefix", func() {
		const (
			secret    = "supersecret"
			jsonBody  = `{"password":"supersecret","padding":"xxxxxxxxxxxxxxxxxxxxxxxx"}`
			jsonLimit = 40
		)
		properties.Set("http.log.response.body.length", jsonLimit)
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, jsonBody)
		}))
		DeferCleanup(server.Close)

		log := logger.New("response-redaction")
		log.SetLogLevel(logger.Trace2)
		response, err := NewClient().WithLogger(log).R(context.Background()).Get(server.URL)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(response.Body.Close)

		Expect(output.String()).To(ContainSubstring("body truncated after 40 bytes"))
		Expect(output.String()).ToNot(ContainSubstring(secret))
	})
})
