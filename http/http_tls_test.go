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

	port := "18080"
	server, err := tlsServer(*serverCrt, port)
	if err != nil {
		t.Fatal(err)
	}
	logger.Infof("Listening on port %s", port)

	serverReady := make(chan struct{})
	go func() {
		close(serverReady)
		err := server.ListenAndServeTLS("", "")
		logger.Infof("server error: %v", err)
	}()

	serverTerminate := make(chan struct{})
	go func() {
		<-serverTerminate
		_ = server.Shutdown(context.Background())
	}()

	<-serverReady
	testData := []struct {
		name      string
		tlsConfig chttp.TLSConfig
	}{
		{"withca", chttp.TLSConfig{CA: string(caPEM)}},
		{"with client certs and CA", chttp.TLSConfig{Cert: string(clientPEM), Key: string(clientKeyPem), CA: string(caPEM)}}, // FIXME: Setup an HTTPs server that requires client auth
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			client, err := chttp.NewClient().TLSConfig(td.tlsConfig)
			if err != nil {
				t.Fatal(err)
			}

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

	serverTerminate <- struct{}{}
}

func tlsServer(cert tls.Certificate, port string) (*http.Server, error) {
	server := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hello, World!"))
		}),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	return server, nil
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
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
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

	// Create tls.Certificate
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Create a new private key PEM
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
