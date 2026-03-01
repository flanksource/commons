package har

import (
	"net/http"
	"sync"

	"github.com/flanksource/commons/http/middlewares"
)

// Collector accumulates HAR entries from multiple sources (main requests,
// OAuth token fetches, redirect hops, retries).
type Collector struct {
	Config  HARConfig
	mu      sync.Mutex
	entries []Entry
}

func NewCollector(cfg HARConfig) *Collector {
	return &Collector{Config: cfg}
}

// Add appends an entry to the collector. Safe for concurrent use.
func (c *Collector) Add(e *Entry) {
	c.mu.Lock()
	c.entries = append(c.entries, *e)
	c.mu.Unlock()
}

// Entries returns a copy of all collected entries.
func (c *Collector) Entries() []Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Entry, len(c.entries))
	copy(out, c.entries)
	return out
}

// Middleware returns a transport middleware that captures each request/response
// into this collector.
func (c *Collector) Middleware() middlewares.Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return capture(req, next, c.Config, c.Add)
		})
	}
}

// Handler returns a func(*Entry) that adds entries to this collector.
// Useful for passing to components that accept a HAR handler callback.
func (c *Collector) Handler() func(*Entry) {
	return c.Add
}
