package certs

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const (
	// CertificateHeader is the first line of a certificate file
	CertificateHeader = "-----BEGIN CERTIFICATE-----"
	// PrivateKeyHeader is the first line of a private key file
	PrivateKeyHeader = "-----BEGIN RSA PRIVATE KEY-----"
)

type CertificateMarshaller struct {
	CertFile       string `yaml:"cert,omitempty"`
	PrivateKeyFile string `yaml:"privateKey,omitempty"`
	Password       string `yaml:"password,omitempty"`
}

func (c *Certificate) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cm CertificateMarshaller
	if err := unmarshal(&cm); err != nil {
		return err
	}

	cert, err := LoadCertificate(cm.CertFile)
	if err != nil {
		return errors.Wrap(err, "failed to load certificate")
	}
	privateKey, err := LoadPrivateKey(cm.PrivateKeyFile)
	if err != nil {
		return errors.Wrap(err, "failed to load private key")
	}

	password := loadPassword(cm.Password)

	certificate, err := DecryptCertificate(cert, privateKey, []byte(password))
	if err != nil {
		return errors.Wrap(err, "failed to decrypt certificate")
	}

	*c = *certificate
	return nil
}

func (c Certificate) MarshalYAML() (interface{}, error) {
	cm := &CertificateMarshaller{
		CertFile:       string(c.EncodedCertificate()),
		PrivateKeyFile: string(c.EncodedPrivateKey()),
	}
	return cm, nil
}

func LoadCertificate(certificate string) ([]byte, error) {
	if strings.HasPrefix(certificate, CertificateHeader) {
		return []byte(certificate), nil
	}

	return loadCertificateBytes(certificate)
}

func LoadPrivateKey(privateKey string) ([]byte, error) {
	if strings.HasPrefix(privateKey, PrivateKeyHeader) {
		return []byte(privateKey), nil
	}

	return loadCertificateBytes(privateKey)
}

func loadCertificateBytes(certificate string) ([]byte, error) {
	if strings.HasPrefix(certificate, "http") || strings.HasPrefix(certificate, "https") {
		resp, err := http.Get(certificate)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to download certificate from url %s", certificate)
		}
		defer resp.Body.Close()
		certBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read response body from url %s", certificate)
		}

		return certBytes, nil
	}

	fullPath, err := filepath.Abs(certificate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand path")
	}

	body, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read certificate %s from disk", certificate)
	}

	return body, nil
}

func loadPassword(password string) string {
	if strings.HasPrefix(password, "$") {
		env := os.Getenv(password[1:])
		if env != "" {
			return env
		}
	}
	return password
}
