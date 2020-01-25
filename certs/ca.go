package certs

import (
	"crypto/rand"
	"crypto/x509"
	"math"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

type CertificateAuthority interface {
	SignCertificate(cert *Certificate, expiryYears int) (*Certificate, error)
	Sign(cert *x509.Certificate, expiry time.Duration) (*x509.Certificate, error)
	GetPublicChain() []*Certificate
}

func (c Certificate) GetPublicChain() []*Certificate {
	return append(c.Chain, &Certificate{X509: c.X509})
}

func (ca *Certificate) SignCertificate(cert *Certificate, expiryYears int) (*Certificate, error) {
	if cert.X509.PublicKey == nil && cert.PrivateKey != nil {
		cert.X509.PublicKey = cert.PrivateKey.Public()
	}
	signed, err := ca.Sign(cert.X509, time.Hour*24*364*time.Duration(expiryYears))
	if err != nil {
		return nil, err
	}

	return &Certificate{
		X509:       signed,
		PrivateKey: cert.PrivateKey,
	}, nil
}

func (ca *Certificate) Sign(cert *x509.Certificate, expiry time.Duration) (*x509.Certificate, error) {
	if cert.SerialNumber == nil {
		serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate random integer for signed certificate")
		}
		cert.SerialNumber = serial
	}

	// Account for clock skew
	cert.NotBefore = time.Now().Add(15 * time.Minute * -1).UTC()
	cert.NotAfter = time.Now().Add(expiry).UTC()
	if cert.KeyUsage == 0 {
		cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	}

	b, err := x509.CreateCertificate(rand.Reader, cert, ca.X509, cert.PublicKey, ca.PrivateKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(b)
}
