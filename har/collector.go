package har

import (
	"errors"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/flanksource/commons/http/middlewares"
)

// Collector accumulates HAR entries from multiple sources (main requests,
// OAuth token fetches, redirect hops, retries). Requests captured through its
// own middlewares are also tracked while in flight, so Entries can show a
// request that has not returned yet.
type Collector struct {
	Config     HARConfig
	mu         sync.Mutex
	entries    []Entry
	pending    map[uint64]inflight
	nextID     uint64
	dropped    int
	handler    func(*Entry)
	onStart    func(*Entry) error
	onComplete func(*Entry) error
	durable    bool
	captureErr error
}

// inflight is a request registered by one of the collector's middlewares that
// has not completed yet.
type inflight struct {
	started time.Time
	entry   Entry
}

func NewCollector(cfg HARConfig) *Collector {
	return &Collector{Config: cfg}
}

// NewCollectorWithHandler creates a collector that retains its own bounded
// view and forwards every completed entry to handler. This lets a
// request-scoped diagnostic collector coexist with a longer-lived HAR export
// collector. In-flight entries are never forwarded.
func NewCollectorWithHandler(cfg HARConfig, handler func(*Entry)) *Collector {
	return &Collector{Config: cfg, handler: handler}
}

// NewCollectorWithLifecycle forwards both states of each request to a durable
// owner. It does not retain completed entries or their bodies in memory.
// Callback errors are available through CaptureError; they do not change the
// HTTP response returned by the wrapped transport.
func NewCollectorWithLifecycle(cfg HARConfig, started, completed func(*Entry) error) *Collector {
	return &Collector{Config: cfg, onStart: started, onComplete: completed, durable: true}
}

func (c *Collector) CaptureError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.captureErr
}

func (c *Collector) recordError(err error) {
	if err == nil {
		return
	}
	c.mu.Lock()
	c.captureErr = errors.Join(c.captureErr, err)
	c.mu.Unlock()
}

// Add appends a completed entry to the collector. Safe for concurrent use.
func (c *Collector) Add(e *Entry) {
	c.complete(0, e)
}

// track registers entry as in flight under a new collector-unique ID and
// returns the function that replaces it with its completed form.
func (c *Collector) track(started time.Time, entry Entry) func(*Entry) {
	c.mu.Lock()
	if c.pending == nil {
		c.pending = map[uint64]inflight{}
	}
	c.nextID++
	id := c.nextID
	entry.ID = strconv.FormatUint(id, 10)
	pendingEntry := entry
	if c.durable && pendingEntry.Request.PostData != nil {
		body := *pendingEntry.Request.PostData
		body.Text = ""
		pendingEntry.Request.PostData = &body
	}
	c.pending[id] = inflight{started: started, entry: pendingEntry}
	onStart := c.onStart
	c.mu.Unlock()
	if onStart != nil {
		entry.Pending = true
		c.recordError(onStart(&entry))
	}
	return func(completed *Entry) {
		completed.ID = entry.ID
		c.complete(id, completed)
	}
}

// complete retires the in-flight registration id (0 for an entry that was
// never tracked) and adds e under the MaxEntries bound in the same critical
// section, so a snapshot never misses the request between the two states.
func (c *Collector) complete(id uint64, e *Entry) {
	c.mu.Lock()
	delete(c.pending, id)
	if c.durable {
		// Durable owners receive the entry through onComplete below.
	} else if c.Config.MaxEntries > 0 && len(c.entries) >= c.Config.MaxEntries {
		c.dropped++
	} else {
		c.entries = append(c.entries, *e)
	}
	handler := c.handler
	onComplete := c.onComplete
	c.mu.Unlock()
	if handler != nil {
		handler(e)
	}
	if onComplete != nil {
		c.recordError(onComplete(e))
	}
}

// Entries returns a copy of the completed entries in completion order,
// followed by a snapshot of the requests still in flight in the order they
// started. An in-flight entry has Pending set and Time/Timings.Wait holding the
// elapsed milliseconds; it does not count toward MaxEntries.
func (c *Collector) Entries() []Entry {
	if c == nil {
		return nil
	}

	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Entry, len(c.entries), len(c.entries)+len(c.pending))
	copy(out, c.entries)
	if c.durable {
		return out
	}
	for _, id := range slices.Sorted(maps.Keys(c.pending)) {
		out = append(out, c.pending[id].snapshot(now))
	}
	return out
}

func (f inflight) snapshot(now time.Time) Entry {
	e := f.entry
	e.Pending = true
	e.Time = millis(now.Sub(f.started))
	e.Timings = Timings{Wait: e.Time}
	return e
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
// into this collector, tracking the request as pending until it completes.
func (c *Collector) Middleware() middlewares.Middleware {
	return c.middleware(capture)
}

// MetadataMiddleware is the collector-backed form of NewMetadataMiddleware:
// headers and timings only, with the request tracked as pending until it
// completes.
func (c *Collector) MetadataMiddleware() middlewares.Middleware {
	return c.middleware(captureMetadata)
}

func (c *Collector) middleware(fn captureFunc) middlewares.Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return fn(req, next, c.Config, c.track)
		})
	}
}

// Handler returns a func(*Entry) that adds entries to this collector.
// Useful for passing to components that accept a HAR handler callback.
func (c *Collector) Handler() func(*Entry) {
	return c.Add
}
