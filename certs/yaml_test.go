package certs

import (
	"io/ioutil"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"gopkg.in/flanksource/yaml.v3"
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

func TestMarshalCertificate(t *testing.T) {
	g := NewWithT(t)

	cfg, err := loadConfig("fixtures/literal.yml")
	g.Expect(err).ToNot(HaveOccurred())

	generatedBytes, err := yaml.Marshal(cfg)
	g.Expect(err).ToNot(HaveOccurred())

	generatedCfg := struct {
		CA *struct {
			Cert       *string `yaml:"cert,omitempty"`
			PrivateKey *string `yaml:"privateKey,omitempty"`
			Password   *string `yaml:"password,omitempty"`
		}
	}{}
	err = yaml.Unmarshal(generatedBytes, &generatedCfg)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(generatedCfg.CA).ToNot(BeNil())
	g.Expect(generatedCfg.CA.Cert).ToNot(BeNil())
	g.Expect(generatedCfg.CA.PrivateKey).ToNot(BeNil())
	g.Expect(generatedCfg.CA.Password).To(BeNil())

	g.Expect(*generatedCfg.CA.Cert).To(Equal(string(cfg.CA.EncodedCertificate())))
	g.Expect(*generatedCfg.CA.PrivateKey).To(Equal(string(cfg.CA.EncodedPrivateKey())))
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
