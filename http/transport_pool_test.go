package http

import (
	"context"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"

	rpchttp "github.com/flanksource/clicky/rpc/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Managed transport pool", func() {
	It("isolates Host and SNI overrides from the pooled TLS configuration", func() {
		server := httptest.NewTLSServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) { _, _ = io.WriteString(w, r.TLS.ServerName) }))
		DeferCleanup(server.Close)
		pool, err := NewTransportPool(TransportPoolOptions{})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		options := TransportOptions{Endpoint: server.URL, TLS: TLSConfig{InsecureSkipVerify: true}}
		client, err := pool.Client(options)
		Expect(err).NotTo(HaveOccurred())
		response, err := client.Header("Host", "alternate.example").R(context.Background()).Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		body, err := response.AsString()
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(Equal("alternate.example"))
		client, err = pool.Client(options)
		Expect(err).NotTo(HaveOccurred())
		response, err = client.R(context.Background()).Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		body, err = response.AsString()
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(BeEmpty())
		client, err = pool.Client(options)
		Expect(err).NotTo(HaveOccurred())
		_, err = client.InsecureSkipVerify(false).Header("Host", "alternate.example").R(context.Background()).Get(server.URL)
		Expect(err).To(HaveOccurred())
	})
	It("uses custom trust roots and preserves request isolation under concurrency", func() {
		server := httptest.NewTLSServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) { _, _ = io.WriteString(w, r.Header.Get("X-Owner")) }))
		DeferCleanup(server.Close)
		ca := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}))
		pool, err := NewTransportPool(TransportPoolOptions{})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		var wg sync.WaitGroup
		for index := 0; index < 8; index++ {
			index := index
			wg.Go(func() {
				defer GinkgoRecover()
				client, err := pool.Client(TransportOptions{Endpoint: server.URL, TLS: TLSConfig{CA: ca}})
				Expect(err).NotTo(HaveOccurred())
				owner := fmt.Sprintf("owner-%d", index)
				client.Header("X-Owner", owner)
				ctx, timings := rpchttp.WithTimings(context.Background())
				response, err := client.R(ctx).Get(server.URL)
				Expect(err).NotTo(HaveOccurred())
				body, err := response.AsString()
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(Equal(owner))
				Expect(timings.Header()).To(ContainSubstring("requests=1"))
			})
		}
		wg.Wait()
	})
	It("reuses connections across separate clients and isolates security generations", func() {
		var connections atomic.Int64
		server := httptest.NewUnstartedServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			_, _ = io.WriteString(w, r.Header.Get("X-Owner"))
		}))
		server.Config.ConnState = func(_ net.Conn, state stdhttp.ConnState) {
			if state == stdhttp.StateNew {
				connections.Add(1)
			}
		}
		server.StartTLS()
		DeferCleanup(server.Close)
		pool, err := NewTransportPool(TransportPoolOptions{Capacity: 1})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		for index, generation := range []string{"first", "first", "rotated", "first"} {
			client, err := pool.Client(TransportOptions{Endpoint: server.URL, SecurityNamespace: generation, TLS: TLSConfig{InsecureSkipVerify: true}})
			Expect(err).NotTo(HaveOccurred())
			client.Header("X-Owner", generation)
			ctx, timings := rpchttp.WithTimings(context.Background())
			response, err := client.R(ctx).Get(server.URL)
			Expect(err).NotTo(HaveOccurred())
			body, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Body.Close()).To(Succeed())
			Expect(string(body)).To(Equal(generation))
			Expect(timings.Header()).To(ContainSubstring("requests=1"))
			if index == 1 {
				Expect(connections.Load()).To(Equal(int64(1)))
			}
		}
		Expect(connections.Load()).To(Equal(int64(3)))
		Expect(pool.Close()).To(Succeed())
		_, err = pool.Client(TransportOptions{Endpoint: server.URL})
		Expect(err).To(MatchError(ContainSubstring("closed")))
	})

	It("rejects invalid pool and TLS configuration", func() {
		_, err := NewTransportPool(TransportPoolOptions{Capacity: -1})
		Expect(err).To(HaveOccurred())
		pool, err := NewTransportPool(TransportPoolOptions{})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(pool.Close)
		_, err = pool.Client(TransportOptions{Endpoint: "https://example.invalid", TLS: TLSConfig{CA: "invalid"}})
		Expect(err).To(HaveOccurred())
	})
})
