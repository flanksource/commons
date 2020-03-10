package config_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/flanksource/commons/config"
	. "github.com/onsi/gomega"
)

type Config struct {
	StringValue config.String `yaml:"stringValue,omitempty"`
}

var (
	tests = map[string]string{
		"plain.yml":       "plain.yml",
		"file_src.yml":    "file_dst.yml",
		"env_var_src.yml": "env_var_dst.yml",
		"http_src.yml":    "http_dst.yml",
		"https_src.yml":   "https_dst.yml",
	}
)

func TestLoadConfig(t *testing.T) {
	g := NewWithT(t)
	os.Setenv("FOO", "bar")

	for src, dst := range tests {
		srcFile := fmt.Sprintf("./fixtures/%s", src)
		dstFile := fmt.Sprintf("./fixtures/%s", dst)
		srcBytes, err := ioutil.ReadFile(srcFile)
		g.Expect(err).ToNot(HaveOccurred())

		dstBytes, err := ioutil.ReadFile(dstFile)
		g.Expect(err).ToNot(HaveOccurred())

		config := &Config{}
		err = yaml.Unmarshal(srcBytes, config)
		g.Expect(err).ToNot(HaveOccurred())

		encodedData, err := yaml.Marshal(config)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(string(encodedData)).To(Equal(string(dstBytes)))
		fmt.Printf("[pass] %s\n", src)
	}
}
