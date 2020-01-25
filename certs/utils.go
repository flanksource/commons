package certs

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"
)

// decodeCertPEM attempts to return a decoded certificate or nil
// if the encoded input does not contain a certificate.
func decodeCertPEM(encoded []byte) (*x509.Certificate, error) {
	if len(encoded) == 0 {
		return nil, fmt.Errorf("empty certificate")
	}
	block, _ := pem.Decode(encoded)
	if block == nil {
		return nil, fmt.Errorf("unable to decode PEM encoded text")
	}

	return x509.ParseCertificate(block.Bytes)
}

func parsePrivateKey(der []byte) (*rsa.PrivateKey, error) {
	key, err := x509.ParsePKCS1PrivateKey(der)
	if err == nil {
		return key, nil
	}

	rsaOrEcKey, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key in either PKCS#1 or PKCS#8 format")
	}
	switch rsaOrEcKey.(type) {
	case *rsa.PrivateKey:
		return rsaOrEcKey.(*rsa.PrivateKey), nil
	}
	return nil, fmt.Errorf("Expecting RSA key, found: %v", reflect.TypeOf(rsaOrEcKey))
}

// decodePrivateKeyPEM attempts to return a decoded key or nil
// if the encoded input does not contain a private key.
func decodePrivateKeyPEM(encoded []byte) (*rsa.PrivateKey, error) {
	if len(encoded) == 0 {
		return nil, fmt.Errorf("empty private key")
	}
	block, _ := pem.Decode(encoded)
	if block == nil {
		return nil, fmt.Errorf("unable to decode PEM encoded text")
	}
	return parsePrivateKey(block.Bytes)
}
