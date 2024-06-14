package http_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"sync"
	"testing"
	"time"

	chttp "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/logger"
)

func TestTLSConfig(t *testing.T) {
	// Generate a self-signed certificate
	certPemData, cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatal(err)
	}

	port := "18080"
	server, err := tlsServer(cert, port)
	if err != nil {
		t.Fatal(err)
	}
	logger.Infof("Listening on port %s", port)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := server.ListenAndServeTLS("", "")
		logger.Infof("server error: %v", err)
		wg.Done()
	}()

	go func() {
		time.Sleep(time.Second)
		_ = server.Shutdown(context.Background())
	}()

	client, err := chttp.NewClient().TLSConfig(chttp.TLSConfig{CA: string(certPemData)})
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

	wg.Wait()
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

func generateSelfSignedCert() ([]byte, tls.Certificate, error) {
	subject := pkix.Name{
		Organization: []string{"Example Company"},
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, tls.Certificate{}, err
	}

	certTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    time.Now(),
		DNSNames:     []string{"localhost"},
		NotAfter:     time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:         true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, tls.Certificate{}, err
	}

	certPEMData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEMData := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	cert, err := tls.X509KeyPair(certPEMData, keyPEMData)
	if err != nil {
		return nil, tls.Certificate{}, err
	}

	return certPEMData, cert, nil
}
