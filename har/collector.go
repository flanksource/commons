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
	dropped int
	handler func(*Entry)
}

func NewCollector(cfg HARConfig) *Collector {
	return &Collector{Config: cfg}
}

// NewCollectorWithHandler creates a collector that retains its own bounded
// view and forwards every entry to handler. This lets a request-scoped
// diagnostic collector coexist with a longer-lived HAR export collector.
func NewCollectorWithHandler(cfg HARConfig, handler func(*Entry)) *Collector {
	return &Collector{Config: cfg, handler: handler}
}

// Add appends an entry to the collector. Safe for concurrent use.
func (c *Collector) Add(e *Entry) {
	c.mu.Lock()
	if c.Config.MaxEntries > 0 && len(c.entries) >= c.Config.MaxEntries {
		c.dropped++
	} else {
		c.entries = append(c.entries, *e)
	}
	handler := c.handler
	c.mu.Unlock()
	if handler != nil {
		handler(e)
	}
}

// Entries returns a copy of all collected entries.
func (c *Collector) Entries() []Entry {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Entry, len(c.entries))
	copy(out, c.entries)
	return out
}

// DroppedEntries returns the number of entries rejected after MaxEntries was
// reached.
func (c *Collector) DroppedEntries() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dropped
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
