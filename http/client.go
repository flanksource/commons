package http

import (
	"io"
	"net/http"

	"github.com/flanksource/commons/logger"
)

// Client is a type that represents an HTTP client
type Client struct {
	httpClient *http.Client
	config     *Config
}

// NewClient configures a new HTTP client using given configuration
func NewClient(config *Config) *Client {
	if config == nil {
		return nil
	}

	if config.Headers == nil {
		config.Headers = map[string]string{}
	}

	if config.Logger == nil {
		config.Logger = logger.StandardLogger()
	}

	return &Client{
		httpClient: createHTTPClient(config),
		config:     config,
	}
}

func createHTTPClient(config *Config) *http.Client {
	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: createHTTPTransport(config),
	}

	return client
}

// Get sends an HTTP GET request
func (c *Client) Get(url string) (*Response, error) {
	request := NewGetRequest(c.config, url)
	return request.Send(c.httpClient, c.config.Logger)
}

// Post sends an HTTP POST request
func (c *Client) Post(url string, contentType string, body io.ReadCloser) (*Response, error) {
	request := NewPostRequest(c.config, url, contentType, body)
	return request.Send(c.httpClient, c.config.Logger)
}

// Patch sends an HTTP PATCH request
func (c *Client) Patch(url string, body io.ReadCloser) (*Response, error) {
	request := NewPatchRequest(c.config, url, body)
	return request.Send(c.httpClient, c.config.Logger)
}

// Put sends an HTTP PUT request
func (c *Client) Put(url string, body io.ReadCloser) (*Response, error) {
	request := NewPutRequest(c.config, url, body)
	return request.Send(c.httpClient, c.config.Logger)
}

// Delete sends an HTTP DELETE request
func (c *Client) Delete(url string) (*Response, error) {
	request := NewDeleteRequest(c.config, url)
	return request.Send(c.httpClient, c.config.Logger)
}
