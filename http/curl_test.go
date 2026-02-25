package http

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/onsi/gomega"
)

func TestToCurl(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("GET with headers", func(t *testing.T) {
		g := gomega.NewWithT(t)
		req, _ := http.NewRequest("GET", "https://example.com/api/data", nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Custom", "value")

		got := ToCurl(req)
		g.Expect(got).To(gomega.ContainSubstring("curl -X GET 'https://example.com/api/data'"))
		g.Expect(got).To(gomega.ContainSubstring("-H 'Accept: application/json'"))
		g.Expect(got).To(gomega.ContainSubstring("-H 'X-Custom: value'"))
		g.Expect(got).ToNot(gomega.ContainSubstring("--data"))
	})

	t.Run("POST with body", func(t *testing.T) {
		g := gomega.NewWithT(t)
		body := `{"key":"value"}`
		req, _ := http.NewRequest("POST", "https://example.com/api", io.NopCloser(bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")

		got := ToCurl(req)
		g.Expect(got).To(gomega.ContainSubstring("curl -X POST"))
		g.Expect(got).To(gomega.ContainSubstring("-H 'Content-Type: application/json'"))
		g.Expect(got).To(gomega.ContainSubstring(`--data '{"key":"value"}'`))
	})

	t.Run("POST body is restored after ToCurl", func(t *testing.T) {
		g := gomega.NewWithT(t)
		body := `hello`
		req, _ := http.NewRequest("POST", "https://example.com", io.NopCloser(bytes.NewBufferString(body)))

		ToCurl(req)

		restored, _ := io.ReadAll(req.Body)
		g.Expect(string(restored)).To(gomega.Equal("hello"))
	})

	t.Run("auth headers are included unredacted", func(t *testing.T) {
		g := gomega.NewWithT(t)
		req, _ := http.NewRequest("GET", "https://example.com", nil)
		req.Header.Set("Authorization", "Bearer secret-token")
		req.Header.Set("Cookie", "session=abc123")

		got := ToCurl(req)
		g.Expect(got).To(gomega.ContainSubstring("-H 'Authorization: Bearer secret-token'"))
		g.Expect(got).To(gomega.ContainSubstring("-H 'Cookie: session=abc123'"))
	})

	t.Run("URL with single quotes is escaped", func(t *testing.T) {
		g = gomega.NewWithT(t)
		req, _ := http.NewRequest("GET", "https://example.com/api?q=it's", nil)
		got := ToCurl(req)
		g.Expect(got).To(gomega.ContainSubstring("'https://example.com/api?q=it'\\''s'"))
	})

	t.Run("nil body produces no --data", func(t *testing.T) {
		g = gomega.NewWithT(t)
		req, _ := http.NewRequest("DELETE", "https://example.com/item/1", nil)
		got := ToCurl(req)
		g.Expect(got).To(gomega.Equal("curl -X DELETE 'https://example.com/item/1'"))
	})
}
