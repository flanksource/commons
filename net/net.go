package net

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/flanksource/commons/logger"
)

func Ping(host string, port int, timeoutSeconds int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), time.Duration(timeoutSeconds)*time.Second)
	if conn != nil {
		conn.Close()
	}
	return err == nil
}

// GET downloads and returns the contents at url
func GET(url string, args ...interface{}) ([]byte, error) {
	url = fmt.Sprintf(url, args...)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
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
		return err
	}
	logger.Tracef("Download %s [%d]-> %s\n", url, resp.StatusCode, path)

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
