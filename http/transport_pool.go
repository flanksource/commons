package http

import (
	"container/list"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type TransportPoolOptions struct {
	// Capacity defaults to 64 transports. Eviction closes idle connections.
	Capacity int
}
type TransportOptions struct {
	Endpoint          string
	SecurityNamespace string
	TLS               TLSConfig
	Proxy             string
}

// TransportPool shares immutable transports, never clients or request middleware.
type TransportPool struct {
	mu       sync.Mutex
	capacity int
	closed   bool
	entries  map[[32]byte]*list.Element
	order    list.List
}

type transportEntry struct {
	key [32]byte
	// transport serves requests and may be a TransportWrapper chain around inner.
	transport stdhttp.RoundTripper
	inner     *stdhttp.Transport
}

func NewTransportPool(options TransportPoolOptions) (*TransportPool, error) {
	if options.Capacity < 0 {
		return nil, errors.New("transport pool capacity must be positive")
	}
	if options.Capacity == 0 {
		options.Capacity = 64
	}
	return &TransportPool{capacity: options.Capacity, entries: make(map[[32]byte]*list.Element)}, nil
}

func (p *TransportPool) Client(options TransportOptions) (*Client, error) {
	endpoint, err := url.Parse(options.Endpoint)
	if err != nil || endpoint.Host == "" || endpoint.User != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, errors.New("transport pool requires an HTTP endpoint without user info")
	}
	options.Endpoint = strings.ToLower(endpoint.Scheme + "://" + endpoint.Host)
	if options.TLS.HandshakeTimeout == 0 {
		options.TLS.HandshakeTimeout = 10 * time.Second
	}
	encoded, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("encode transport settings: %w", err)
	}
	entry, err := p.transport(sha256.Sum256(encoded), options)
	if err != nil {
		return nil, err
	}
	client := NewClient()
	client.httpClient.Transport = entry.transport
	client.tlsConfig = entry.inner.TLSClientConfig
	return client, nil
}

func (p *TransportPool) transport(key [32]byte, options TransportOptions) (transportEntry, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return transportEntry{}, errors.New("transport pool is closed")
	}
	if entry := p.entries[key]; entry != nil {
		p.order.MoveToFront(entry)
		return entry.Value.(transportEntry), nil
	}
	client, err := NewClient().TLSConfig(options.TLS)
	if err != nil {
		return transportEntry{}, err
	}
	if options.Proxy != "" {
		proxy, err := url.Parse(options.Proxy)
		if err != nil || proxy.Scheme == "" || proxy.Host == "" {
			return transportEntry{}, errors.New("transport pool proxy must be an absolute URL")
		}
		if err := client.configureTransport(func(transport *stdhttp.Transport) {
			transport.Proxy = stdhttp.ProxyURL(proxy)
		}); err != nil {
			return transportEntry{}, fmt.Errorf("configure transport pool proxy: %w", err)
		}
	}
	inner, _, err := unwrapTransport(client.httpClient.Transport)
	if err != nil {
		return transportEntry{}, err
	}
	entry := transportEntry{key: key, transport: client.httpClient.Transport, inner: inner}
	p.entries[key] = p.order.PushFront(entry)
	if p.order.Len() > p.capacity {
		old := p.order.Back()
		evicted := old.Value.(transportEntry)
		delete(p.entries, evicted.key)
		p.order.Remove(old)
		evicted.inner.CloseIdleConnections()
	}
	return entry, nil
}

func (p *TransportPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	for _, element := range p.entries {
		element.Value.(transportEntry).inner.CloseIdleConnections()
	}
	clear(p.entries)
	p.order.Init()
	return nil
}
