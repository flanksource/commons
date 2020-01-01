package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Certificate is a X509 certifcate / private key pair
type Certificate struct {
	X509       *x509.Certificate
	PrivateKey *rsa.PrivateKey
	Chain      []*Certificate
}

// EncodedPrivateKey returns PEM-encoded private key data.
func (c Certificate) EncodedPrivateKey() []byte {
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(c.PrivateKey),
	}
	return pem.EncodeToMemory(&block)
}

// EncodedPublicKey returns PEM-encoded public key data.
func (c Certificate) EncodedPublicKey() []byte {

	publicKey := c.PrivateKey.PublicKey

	der, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		panic(err)
	}

	if len(der) == 0 {
		panic("nil pub key")
	}

	block := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}
	return pem.EncodeToMemory(&block)
}

// NewCertificate decodes a certificate / private key pair and returns a Certificate
func DecodeCertificate(cert []byte, privateKey []byte) (*Certificate, error) {
	x509, err := decodeCertPEM(cert)
	if err != nil {
		return nil, err
	}
	key, err := decodePrivateKeyPEM(privateKey)
	if err != nil {
		return nil, err
	}
	return &Certificate{
		PrivateKey: key,
		X509:       x509,
	}, nil
}

// GetHash returns the encoded sha256 hash for the certificate
func (c Certificate) GetHash() (string, error) {
	certHash := sha256.Sum256(c.X509.RawSubjectPublicKeyInfo)
	return "sha256:" + strings.ToLower(hex.EncodeToString(certHash[:])), nil
}

// EncodedCertificate returns PEM-endcoded certificate data.
func (c Certificate) EncodedCertificate() []byte {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.X509.Raw,
	}
	return pem.EncodeToMemory(&block)
}

type CertificateAuthority interface {
	SignCertificate(cert *Certificate, expiryYears int) (*Certificate, error)
	Sign(cert *x509.Certificate, expiry time.Duration) (*x509.Certificate, error)
	GetPublicChain() []*Certificate
}

func (ca *Certificate) SignCertificate(cert *Certificate, expiryYears int) (*Certificate, error) {
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

	b, err := x509.CreateCertificate(rand.Reader, cert, ca.X509, ca.PrivateKey.Public(), ca.PrivateKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(b)
}

// decodeCertPEM attempts to return a decoded certificate or nil
// if the encoded input does not contain a certificate.
func decodeCertPEM(encoded []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(encoded)
	if block == nil {
		return nil, nil
	}

	return x509.ParseCertificate(block.Bytes)
}

// decodePrivateKeyPEM attempts to return a decoded key or nil
// if the encoded input does not contain a private key.
func decodePrivateKeyPEM(encoded []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(encoded)
	if block == nil {
		return nil, nil
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func (c *Certificate) AsTLSSecret() map[string][]byte {
	return map[string][]byte{
		"tls.crt": c.EncodedCertificate(),
		"tls.key": c.EncodedPrivateKey(),
	}
}
