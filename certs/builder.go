package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"
)

type CertificateBuilder struct {
	*Certificate
}

func NewCertificateBuilder(commonName string) *CertificateBuilder {
	b := &CertificateBuilder{}
	b.Certificate = &Certificate{}
	b.Certificate.X509 = &x509.Certificate{
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	b.Certificate.PrivateKey = key
	b.X509.PublicKey = key.Public()
	return b
}

func (b *CertificateBuilder) Organization(org string) *CertificateBuilder {
	b.Certificate.X509.Subject.Organization = []string{org}
	return b
}

func (b *CertificateBuilder) OrganizationUnit(ou string) *CertificateBuilder {
	b.Certificate.X509.Subject.OrganizationalUnit = []string{ou}
	return b
}

func (b *CertificateBuilder) AltName(names ...string) *CertificateBuilder {
	b.Certificate.X509.DNSNames = append(b.Certificate.X509.DNSNames, names...)
	return b
}

func (b *CertificateBuilder) Server() *CertificateBuilder {
	b.Certificate.X509.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	return b
}

func (b *CertificateBuilder) Client() *CertificateBuilder {
	b.Certificate.X509.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	return b
}

func (b *CertificateBuilder) CA() *CertificateBuilder {
	b.Certificate.X509.KeyUsage = b.Certificate.X509.KeyUsage | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	b.Certificate.X509.MaxPathLenZero = true
	b.Certificate.X509.BasicConstraintsValid = true
	b.Certificate.X509.MaxPathLen = 0
	b.Certificate.X509.IsCA = true
	return b
}

func (b *CertificateBuilder) ValidYears(years int) *CertificateBuilder {
	b.Certificate.X509.NotAfter = time.Now().Add(time.Duration(years) * time.Hour * 24 * 365)
	b.Certificate.X509.NotBefore = time.Now().Add(-2 * time.Hour)
	return b
}
