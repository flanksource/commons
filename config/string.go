package config

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/flanksource/commons/files"
	getter "github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
)

type String string

func NewString(rawValue string) (String, error) {
	if strings.HasPrefix(rawValue, "file://") {
		filename := rawValue[7:]
		value, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read file %s", filename)
		}
		return String(value), nil
	}

	if strings.HasPrefix(rawValue, "$") {
		envVar := rawValue[1:]
		value := os.Getenv(envVar)
		if value == "" {
			return "", errors.Errorf("failed to get env variable %s", envVar)
		}
		return String(value), nil
	}

	if strings.HasPrefix(rawValue, "url://") {
		url := rawValue[6:]
		resp, err := http.Get(url)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get URL %s", url)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read response body for URL %s", url)
		}
		return String(body), nil
	}

	if strings.HasPrefix(rawValue, "go-getter://") {
		url := rawValue[12:]
		tmpfile := files.TempFileName("go-getter-download", "")
		defer os.Remove(tmpfile)
		client := &getter.Client{
			Ctx:     context.TODO(),
			Src:     url,
			Dst:     tmpfile,
			Mode:    getter.ClientModeFile,
			Options: []getter.ClientOption{},
		}
		if err := client.Get(); err != nil {
			return "", errors.Wrapf(err, "failed to get URL %s", url)
		}
		body, err := ioutil.ReadFile(tmpfile)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read tempfile %s", tmpfile)
		}
		return String(body), nil
	}

	return String(rawValue), nil
}

func (s *String) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var sm string
	if err := unmarshal(&sm); err != nil {
		return err
	}

	sv, err := NewString(sm)
	if err != nil {
		return errors.Wrapf(err, "failed to load string value %s", sm)
	}

	*s = sv
	return nil
}
