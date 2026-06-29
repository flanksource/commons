package http_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	chttp "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/logger"
	"github.com/samber/lo"
)

func TestTLSConfig(t *testing.T) {
	caX509, caCrt, caPEM, _, err := createCert(nil, nil, "Flanksource")
	if err != nil {
		t.Fatal(err)
	}

	_, serverCrt, _, _, err := createCert(caX509, caCrt.PrivateKey, "localhost")
	if err != nil {
		t.Fatal(err)
	}

	_, _, clientPEM, clientKeyPem, err := createCert(caX509, caCrt.PrivateKey, "client")
	if err != nil {
		t.Fatal(err)
	}

	_, _, badClientPEM, badClientKeyPem, err := createCert(nil, nil, "bad-client")
	if err != nil {
		t.Fatal(err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM) {
		t.Fatal(err)
	}

	testData := []struct {
		name      string
		expectErr bool
		serverTLS *tls.Config
		clientTLS chttp.TLSConfig
	}{
		{
			name:      "Client provides CA",
			clientTLS: chttp.TLSConfig{CA: string(caPEM)},
			serverTLS: &tls.Config{
				Certificates: []tls.Certificate{*serverCrt},
			},
		},
		{
			name:      "Client doesn't provide CA",
			clientTLS: chttp.TLSConfig{},
			expectErr: true,
			serverTLS: &tls.Config{
				Certificates: []tls.Certificate{*serverCrt},
			},
		},
		{
			name: "mTLS | client provides client certs",
			clientTLS: chttp.TLSConfig{
				Cert: string(clientPEM),
				Key:  string(clientKeyPem),
				CA:   string(caPEM),
			},
			serverTLS: &tls.Config{
				Certificates: []tls.Certificate{*serverCrt},
				ClientCAs:    certPool,
				ClientAuth:   tls.RequireAndVerifyClientCert,
			},
		},
		{
			name: "mTLS | client doesn't provides client certs",
			clientTLS: chttp.TLSConfig{
				CA: string(caPEM),
			},
			expectErr: true,
			serverTLS: &tls.Config{
				Certificates: []tls.Certificate{*serverCrt},
				ClientCAs:    certPool,
				ClientAuth:   tls.RequireAndVerifyClientCert,
			},
		},
		{
			name:      "mTLS | client provides bad certs",
			expectErr: true,
			clientTLS: chttp.TLSConfig{
				CA:   string(caPEM),
				Cert: string(badClientPEM),
				Key:  string(badClientKeyPem),
			},
			serverTLS: &tls.Config{
				Certificates: []tls.Certificate{*serverCrt},
				ClientCAs:    certPool,
				ClientAuth:   tls.RequireAndVerifyClientCert,
			},
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			port := "18080"
			defer startTLSServer(t, port, td.serverTLS)()

			client, err := chttp.NewClient().TLSConfig(td.clientTLS)
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.BaseURL(fmt.Sprintf("https://localhost:%s", port)).R(context.Background()).Get("/")
			if err != nil {
				if !td.expectErr {
					t.Fatal(err)
				}
				return
			} else {
				if td.expectErr {
					t.Fatal("expected error")
				}
			}

			r, err := response.AsString()
			if err != nil {
				t.Fatal(err)
			}
			if r != "Hello, World!" {
				t.Fatal(r)
			}
		})
	}
}

// startTLSServer binds the listener synchronously before serving so the port is
// guaranteed to accept connections by the time it returns. It returns a cleanup
// function that shuts the server down.
func startTLSServer(t *testing.T, port string, tlsConfig *tls.Config) func() {
	t.Helper()

	listener, err := net.Listen("tcp", net.JoinHostPort("", port))
	if err != nil {
		t.Fatal(err)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hello, World!"))
		}),
		TLSConfig: tlsConfig,
	}

	go func() {
		if err := server.ServeTLS(listener, "", ""); err != nil && err != http.ErrServerClosed {
			logger.Infof("server error: %v", err)
		}
	}()

	return func() { _ = server.Shutdown(context.Background()) }
}

func createCert(parent *x509.Certificate, signerKey any, cn string) (*x509.Certificate, *tls.Certificate, []byte, []byte, error) {
	isCa := parent == nil
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName:    cn,
			Organization:  []string{"Example Company"},
			Country:       []string{"US"},
			Province:      []string{"CA"},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"1600 Amphitheatre Pkwy"},
			PostalCode:    []string{"94043"},
		},
		DNSNames:              []string{cn},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  isCa,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if isCa {
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	} else {
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
		template.BasicConstraintsValid = false
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	template.SerialNumber = serialNumber

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, lo.CoalesceOrEmpty(parent, template), &privateKey.PublicKey, lo.CoalesceOrEmpty[any](signerKey, privateKey))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	pemBlock = &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}
	privateKeyBytes := pem.EncodeToMemory(pemBlock)

	certificate, err := tls.X509KeyPair(pemBytes, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return template, &certificate, pemBytes, privateKeyBytes, nil
}

// expiredServerTLS returns a *tls.Config holding a self-signed certificate that
// already expired, for exercising InsecureSkipVerify against an invalid cert
// without depending on an external host.
func expiredServerTLS(t *testing.T) *tls.Config {
	t.Helper()

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-48 * time.Hour),
		NotAfter:     time.Now().Add(-24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
	if err != nil {
		t.Fatal(err)
	}

	return &tls.Config{Certificates: []tls.Certificate{cert}}
}

func TestTLSLogging(t *testing.T) {
	// Enable trace logging to see TLS output
	logger.StandardLogger().SetLogLevel(5)
	
	caX509, caCrt, caPEM, _, err := createCert(nil, nil, "Flanksource")
	if err != nil {
		t.Fatal(err)
	}

	_, serverCrt, _, _, err := createCert(caX509, caCrt.PrivateKey, "localhost")
	if err != nil {
		t.Fatal(err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM) {
		t.Fatal(err)
	}

	t.Run("TLS logging with valid certificate", func(t *testing.T) {
		port := "18090"
		defer startTLSServer(t, port, &tls.Config{
			Certificates: []tls.Certificate{*serverCrt},
		})()

		client, err := chttp.NewClient().TLSConfig(chttp.TLSConfig{CA: string(caPEM)})
		if err != nil {
			t.Fatal(err)
		}
		client = client.WithHttpLogging(5, 7)

		logger.Infof("\n=== Making HTTPS request to test TLS logging ===")
		response, err := client.BaseURL(fmt.Sprintf("https://localhost:%s", port)).R(context.Background()).Get("/")
		if err != nil {
			t.Fatal(err)
		}

		r, err := response.AsString()
		if err != nil {
			t.Fatal(err)
		}
		if r != "Hello, World!" {
			t.Fatal(r)
		}
	})
}
