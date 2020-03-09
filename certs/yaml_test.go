package certs

import (
	"io/ioutil"
	"testing"

	"github.com/pkg/errors"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

type exampleConfig struct {
	CA Certificate `yaml:"ca"`
}

func TestLoadCertificateFromFiles(t *testing.T) {
	g := NewWithT(t)
	cfg, err := loadConfig("fixtures/file.yml")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cfg.CA.X509.Subject.CommonName).To(Equal("k8s"))
}

func TestLoadCertificateFromURL(t *testing.T) {
	g := NewWithT(t)
	cfg, err := loadConfig("fixtures/remote.yml")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cfg.CA.X509.Subject.CommonName).To(Equal("*.test.google.com.au"))
}

func TestLoadCertificateFromLiteral(t *testing.T) {
	g := NewWithT(t)
	cfg, err := loadConfig("fixtures/literal.yml")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cfg.CA.X509.Subject.CommonName).To(Equal("wildcard.literal.flanksource.com"))
}

func loadConfig(path string) (*exampleConfig, error) {
	cfgBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", path)
	}

	cfg := exampleConfig{}
	err = yaml.Unmarshal(cfgBytes, &cfg)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse yml for file %s", path)
	}

	return &cfg, nil
}
