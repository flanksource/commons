package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"sync"

	"github.com/flanksource/commons/http/middlewares"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type staticResolver map[string][]net.IPAddr

func (r staticResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	addresses, ok := r[host]
	if !ok {
		return nil, fmt.Errorf("no address for %s", host)
	}
	return addresses, nil
}

type recordingDialer struct {
	mu        sync.Mutex
	addresses []string
}

func (d *recordingDialer) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	d.mu.Lock()
	d.addresses = append(d.addresses, address)
	d.mu.Unlock()
	return nil, nil
}

func (d *recordingDialer) Addresses() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.addresses...)
}

var _ = Describe("Defensive HTTP client", func() {
	It("uses the public-only, no-proxy, no-redirect defaults", func() {
		client := NewDefensive()

		Expect(client.maxRedirects).To(BeZero())
		configured, ok := client.httpClient.Transport.(*defensiveRoundTripper)
		Expect(ok).To(BeTrue())
		Expect(configured.options).To(Equal(DefensiveOptions{Public: true}))

		base := stdhttp.DefaultTransport.(*stdhttp.Transport).Clone()
		wrapped := Defensive(DefensiveOptions{Public: true})(base)
		transport, ok := wrapped.(*defensiveRoundTripper)
		Expect(ok).To(BeTrue())
		Expect(transport.transport).ToNot(BeIdenticalTo(base))
		Expect(transport.transport.Proxy).To(BeNil())
		Expect(transport.transport.TLSClientConfig).ToNot(BeNil())
		Expect(transport.transport.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
	})

	It("composes as middleware around the actual client transport", func() {
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			_, _ = w.Write([]byte("ok"))
		}))
		DeferCleanup(server.Close)

		observed := false
		observe := func(next stdhttp.RoundTripper) stdhttp.RoundTripper {
			return middlewares.RoundTripperFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
				observed = true
				return next.RoundTrip(req)
			})
		}
		client := NewClient().Use(observe, Defensive(DefensiveOptions{Localhost: true}))

		response, err := client.R(context.Background()).Get(server.URL)

		Expect(err).ToNot(HaveOccurred())
		Expect(response.Body.Close()).To(Succeed())
		Expect(observed).To(BeTrue())
	})

	It("replaces custom TLS dialing with the validated dial path", func() {
		dialStopped := errors.New("dial stopped")
		baseDialed := false
		customTLSDialed := false
		base := stdhttp.DefaultTransport.(*stdhttp.Transport).Clone()
		base.DialContext = func(context.Context, string, string) (net.Conn, error) {
			baseDialed = true
			return nil, dialStopped
		}
		base.DialTLSContext = func(context.Context, string, string) (net.Conn, error) {
			customTLSDialed = true
			return nil, nil
		}
		transport := Defensive(DefensiveOptions{Public: true})(base).(*defensiveRoundTripper)

		_, err := transport.transport.DialTLSContext(context.Background(), "tcp", "93.184.216.34:443")

		Expect(errors.Is(err, dialStopped)).To(BeTrue())
		Expect(baseDialed).To(BeTrue())
		Expect(customTLSDialed).To(BeFalse())
	})

	DescribeTable("classifies resolved addresses by policy",
		func(raw string, options DefensiveOptions, allowed bool) {
			err := validateDefensiveIP(net.ParseIP(raw), options)
			if allowed {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
		Entry("public", "93.184.216.34", DefensiveOptions{Public: true}, true),
		Entry("public disabled", "93.184.216.34", DefensiveOptions{}, false),
		Entry("private", "10.10.0.8", DefensiveOptions{Private: true}, true),
		Entry("carrier-grade private", "100.64.1.8", DefensiveOptions{Private: true}, true),
		Entry("IPv6 private", "fd00::8", DefensiveOptions{Private: true}, true),
		Entry("localhost", "127.0.0.1", DefensiveOptions{Localhost: true}, true),
		Entry("IPv6 localhost", "::1", DefensiveOptions{Localhost: true}, true),
		Entry("metadata", "100.100.100.200", DefensiveOptions{Public: true, Private: true, Localhost: true}, false),
		Entry("link-local", "169.254.169.254", DefensiveOptions{Public: true, Private: true, Localhost: true}, false),
		Entry("unspecified", "0.0.0.0", DefensiveOptions{Public: true, Private: true, Localhost: true}, false),
		Entry("multicast", "224.0.0.1", DefensiveOptions{Public: true, Private: true, Localhost: true}, false),
	)

	It("rejects every resolution when one address is disallowed", func() {
		dialer := &recordingDialer{}
		dial := defensiveDialContext(
			DefensiveOptions{Public: true},
			staticResolver{"mixed.example": {
				{IP: net.ParseIP("93.184.216.34")},
				{IP: net.ParseIP("10.0.0.4")},
			}},
			dialer,
		)

		_, err := dial(context.Background(), "tcp", "mixed.example:443")

		Expect(err).To(MatchError(ContainSubstring("10.0.0.4")))
		Expect(dialer.Addresses()).To(BeEmpty())
	})

	It("dials the approved resolved address instead of resolving again", func() {
		dialer := &recordingDialer{}
		dial := defensiveDialContext(
			DefensiveOptions{Public: true},
			staticResolver{"public.example": {{IP: net.ParseIP("93.184.216.34")}}},
			dialer,
		)

		_, err := dial(context.Background(), "tcp", "public.example:443")

		Expect(err).ToNot(HaveOccurred())
		Expect(dialer.Addresses()).To(Equal([]string{"93.184.216.34:443"}))
	})

	It("does not follow redirects unless explicitly enabled", func() {
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			if r.URL.Path == "/redirect" {
				stdhttp.Redirect(w, r, "/result", stdhttp.StatusFound)
				return
			}
			_, _ = w.Write([]byte("followed"))
		}))
		DeferCleanup(server.Close)

		_, err := NewClient().Use(Defensive(DefensiveOptions{Localhost: true})).R(context.Background()).Get(server.URL + "/redirect")
		Expect(err).To(MatchError(ContainSubstring("redirect")))

		followed, err := NewClient().Use(Defensive(DefensiveOptions{Localhost: true, Redirect: true})).R(context.Background()).Get(server.URL + "/redirect")
		Expect(err).ToNot(HaveOccurred())
		body, err := followed.AsString()
		Expect(err).ToNot(HaveOccurred())
		Expect(body).To(Equal("followed"))
	})

	It("fails every request when it does not wrap the concrete transport", func() {
		hidden := middlewares.RoundTripperFunc(func(*stdhttp.Request) (*stdhttp.Response, error) {
			return nil, errors.New("should never be reached")
		})

		wrapped := Defensive(DefensiveOptions{Public: true})(hidden)

		Expect(wrapped).To(BeAssignableToTypeOf(misconfiguredRoundTripper{}))
		_, err := wrapped.RoundTrip(httptest.NewRequest(stdhttp.MethodGet, "https://example.com", nil))
		Expect(err).To(MatchError(ContainSubstring("must wrap an *http.Transport")))
	})

	Describe("proxied destinations", func() {
		It("delegates hostname resolution to an explicitly enabled proxy", func() {
			dialStopped := errors.New("proxy dial stopped")
			proxy, err := url.Parse("http://93.184.216.34:3128")
			Expect(err).ToNot(HaveOccurred())
			base := stdhttp.DefaultTransport.(*stdhttp.Transport).Clone()
			base.Proxy = stdhttp.ProxyURL(proxy)
			base.DialContext = func(context.Context, string, string) (net.Conn, error) {
				return nil, dialStopped
			}
			transport := Defensive(DefensiveOptions{Public: true, Proxy: true})(base)

			_, err = transport.RoundTrip(httptest.NewRequest(stdhttp.MethodGet, "https://tenant.example/data", nil))

			Expect(errors.Is(err, dialStopped)).To(BeTrue())
		})

		DescribeTable("refuses proxy credentials without proxy TLS",
			func(rawProxy string) {
				const password = "proxy-secret"
				proxy, err := url.Parse(rawProxy)
				Expect(err).ToNot(HaveOccurred())
				base := stdhttp.DefaultTransport.(*stdhttp.Transport).Clone()
				base.Proxy = stdhttp.ProxyURL(proxy)
				base.DialContext = func(context.Context, string, string) (net.Conn, error) {
					return nil, errors.New("proxy dial should not be reached")
				}
				transport := Defensive(DefensiveOptions{Public: true, Proxy: true})(base)

				_, err = transport.RoundTrip(httptest.NewRequest(stdhttp.MethodGet, "https://93.184.216.35/data", nil))

				Expect(err).To(MatchError(ContainSubstring("proxy credentials")))
				Expect(err.Error()).ToNot(ContainSubstring(password))
			},
			Entry("HTTP proxy", "http://alice:proxy-secret@93.184.216.34:3128"),
			Entry("scheme-less proxy", "//alice:proxy-secret@93.184.216.34:3128"),
			Entry("SOCKS proxy", "socks5://alice:proxy-secret@93.184.216.34:1080"),
		)

		It("allows proxy credentials over HTTPS", func() {
			dialStopped := errors.New("proxy dial stopped")
			proxy, err := url.Parse("https://alice:proxy-secret@93.184.216.34:3128")
			Expect(err).ToNot(HaveOccurred())
			base := stdhttp.DefaultTransport.(*stdhttp.Transport).Clone()
			base.Proxy = stdhttp.ProxyURL(proxy)
			base.DialContext = func(context.Context, string, string) (net.Conn, error) {
				return nil, dialStopped
			}
			transport := Defensive(DefensiveOptions{Public: true, Proxy: true})(base)

			_, err = transport.RoundTrip(httptest.NewRequest(stdhttp.MethodGet, "https://93.184.216.35/data", nil))

			Expect(errors.Is(err, dialStopped)).To(BeTrue())
		})
	})

	Describe("credentials", func() {
		It("refuses to send an Authorization header over cleartext http", func() {
			request := httptest.NewRequest(stdhttp.MethodGet, "http://public.example/data", nil)
			request.Header.Set("Authorization", "Bearer super-secret")

			err := validateDefensiveCredentials(request)

			Expect(err).To(MatchError(ContainSubstring("Authorization")))
			Expect(err.Error()).ToNot(ContainSubstring("super-secret"))
		})

		It("allows credentials over https and over loopback", func() {
			secure := httptest.NewRequest(stdhttp.MethodGet, "https://public.example/data", nil)
			secure.Header.Set("Authorization", "Bearer super-secret")
			Expect(validateDefensiveCredentials(secure)).To(Succeed())

			loopback := httptest.NewRequest(stdhttp.MethodGet, "http://127.0.0.1:8080/data", nil)
			loopback.Header.Set("Authorization", "Bearer super-secret")
			Expect(validateDefensiveCredentials(loopback)).To(Succeed())
		})

		It("still authenticates against a loopback test server", func() {
			server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				user, _, _ := r.BasicAuth()
				_, _ = w.Write([]byte(user))
			}))
			DeferCleanup(server.Close)

			response, err := NewClient().
				Use(Defensive(DefensiveOptions{Localhost: true})).
				Auth("alice", "hunter2").
				R(context.Background()).Get(server.URL)

			Expect(err).ToNot(HaveOccurred())
			body, err := response.AsString()
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(Equal("alice"))
		})
	})

	It("revalidates an enabled redirect before following it", func() {
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			stdhttp.Redirect(w, r, "http://169.254.169.254/latest/meta-data", stdhttp.StatusFound)
		}))
		DeferCleanup(server.Close)

		_, err := NewClient().Use(Defensive(DefensiveOptions{Localhost: true, Redirect: true})).R(context.Background()).Get(server.URL)

		Expect(err).To(MatchError(ContainSubstring("169.254.169.254")))
	})
})
