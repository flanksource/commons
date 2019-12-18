package net

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

type API struct {
	Client  *http.Client
	baseURL string
}

// NewClient returns new client ...
func NewClient(url string, args ...interface{}) *API {
	url = fmt.Sprintf(url, args...)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	return &API{
		Client:  client,
		baseURL: url,
	}
}

// GET downloads and returns the contents at url
func (a *API) GET() ([]byte, error) {
	resp, err := a.Client.Get(a.baseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return body, fmt.Errorf(resp.Status)
	}
	return body, nil
}

// Download the url to the path on disk
func Download(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	log.Debugf("Download %s [%d]-> %s\n", url, resp.StatusCode, path)

	if resp.StatusCode != 200 {
		return fmt.Errorf(resp.Status)
	}
	defer resp.Body.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
