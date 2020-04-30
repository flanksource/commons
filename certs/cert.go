package certs

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// Certificate is a X509 certifcate / private key pair
type Certificate struct {
	X509       *x509.Certificate
	PrivateKey *rsa.PrivateKey
	Chain      []*Certificate
}

// DecryptCertificate decrypts a certificate / private key pair and returns a Certificate
func DecryptCertificate(cert []byte, privateKey []byte, password []byte) (*Certificate, error) {
	if len(password) == 0 {
		logrus.Warnf("No password provided for CA certificate")
		return DecodeCertificate(cert, privateKey)
	}

	var err error
	var key *rsa.PrivateKey
	block, _ := pem.Decode(privateKey)

	var decrypted []byte
	if decrypted, err = x509.DecryptPEMBlock(block, password); err != nil {
		return nil, err
	}
	if key, err = parsePrivateKey(decrypted); err != nil {
		return nil, err
	}

	x509, err := decodeCertPEM(cert)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		PrivateKey: key,
		X509:       x509,
	}, nil
}

// DecodeCertificate decodes a certificate / private key pair and returns a Certificate
func DecodeCertificate(cert []byte, privateKey []byte) (*Certificate, error) {
	x509, err := decodeCertPEM(cert)
	if err != nil {
		return nil, fmt.Errorf("cannot decode certificate %v", err)
	}

	key, err := decodePrivateKeyPEM(privateKey)
	if err != nil {
		return nil, fmt.Errorf("cannot decode private key %v", err)
	}
	return &Certificate{
		PrivateKey: key,
		X509:       x509,
	}, nil
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

func (c *Certificate) AsTLSSecret() map[string][]byte {
	return map[string][]byte{
		"tls.crt": c.EncodedCertificate(),
		"tls.key": c.EncodedPrivateKey(),
	}
}

func (c *Certificate) AsTLSConfig() *tls.Config) {
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(c.EncodedCertificate())
	return &tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{c.X509},
	}
}
