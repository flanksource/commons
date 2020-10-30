package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/flanksource/commons/logger"
)

type Request struct {
	config *Config
	verb   string
	url    string
	body   io.ReadCloser
}

type Requester interface {
	Send(client *http.Client, logger logger.Logger) (*Response, error)
}

type ContentType struct {
	contentType string
}

type GetRequest struct {
	*Request
}

type PostRequest struct {
	*Request
	*ContentType
}

type PutRequest struct {
	*Request
}

type PatchRequest struct {
	*Request
}

type DeleteRequest struct {
	*Request
}

// NewGetRequest creates a new HTTP GET request
func NewGetRequest(config *Config, url string) *GetRequest {
	return &GetRequest{
		&Request{
			verb:   "GET",
			url:    url,
			config: config,
		},
	}
}

// NewPostRequest creates a new HTTP POST request
func NewPostRequest(config *Config, url string, contentType string, body io.ReadCloser) *PostRequest {
	return &PostRequest{
		&Request{
			verb:   "POST",
			url:    url,
			body:   body,
			config: config,
		},
		&ContentType{
			contentType: contentType,
		},
	}
}

// NewPutRequest creates a new HTTP PUT request
func NewPutRequest(config *Config, url string, body io.ReadCloser) *PutRequest {
	return &PutRequest{
		&Request{
			verb:   "PUT",
			url:    url,
			body:   body,
			config: config,
		},
	}
}

// NewPatchRequest creates a new HTTP PATCH request
func NewPatchRequest(config *Config, url string, body io.ReadCloser) *PatchRequest {
	return &PatchRequest{
		&Request{
			verb:   "PATCH",
			url:    url,
			body:   body,
			config: config,
		},
	}
}

// NewDeleteRequest creates a new HTTP DELETE request
func NewDeleteRequest(config *Config, url string) *DeleteRequest {
	return &DeleteRequest{
		&Request{
			verb:   "DELETE",
			url:    url,
			config: config,
		},
	}
}

// Send the HTTP GET request
func (r *GetRequest) Send(client *http.Client, logger logger.Logger) (*Response, error) {
	// GET requests are idempotent so can have retries
	var retries uint = r.config.Retries
	return r.sendRequest(client, logger, retries)
}

// Send the HTTP POST request
func (r *PostRequest) Send(client *http.Client, logger logger.Logger) (*Response, error) {
	r.config.Headers["Content-Type"] = r.contentType

	// POST is non-idempotent so can have no retries
	var retries uint = 0
	return r.sendRequest(client, logger, retries)
}

// Send the HTTP PUT request
func (r *PutRequest) Send(client *http.Client, logger logger.Logger) (*Response, error) {
	// PUT is non-idempotent so can have no retries
	var retries uint = 0
	return r.sendRequest(client, logger, retries)
}

// Send the HTTP PATCH request
func (r *PatchRequest) Send(client *http.Client, logger logger.Logger) (*Response, error) {
	// PATCH is non-idempotent so can have no retries
	var retries uint = 0
	return r.sendRequest(client, logger, retries)
}

// Send the HTTP DELETE request
func (r *DeleteRequest) Send(client *http.Client, logger logger.Logger) (*Response, error) {
	// DELETE is non-idempotent so can have no retries
	var retries uint = 0
	return r.sendRequest(client, logger, retries)
}

// createHTTPRequest configures an HTTP request with the configured values
func (r *Request) createHTTPRequest() (*http.Request, error) {
	requestURL := r.url
	if baseURL := strings.TrimSpace(r.config.BaseURL); baseURL != "" {
		requestURL = fmt.Sprintf("%s/%s", baseURL, r.url)
	}

	request, err := http.NewRequest(r.verb, requestURL, r.body)
	if err != nil {
		return nil, err
	}

	// apply headers
	for key, value := range r.config.Headers {
		request.Header.Set(key, value)
	}

	return request, nil
}

// sendRequest sends the request using the given HTTP client
func (r *Request) sendRequest(client *http.Client, logger logger.Logger, retriesRemaining uint) (*Response, error) {
	request, err := r.createHTTPRequest()
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {

		// if the retries have been exhausted (or not configured), bail out
		if retriesRemaining <= 0 {
			return nil, err
		}

		retriesRemaining--

		backoffTime := exponentialBackoff(r.config, retriesRemaining)
		logger.Warnf("backing off for %v before next retry", backoffTime)

		return r.sendRequest(client, logger, retriesRemaining)
	}

	return &Response{response}, nil
}

type RequestLoggableStrings struct {
	Headers string
	Body    string
}

// GetLoggableStrings returns the Headers and Body of the response as strings that can be logged while
// maintaining the request body's readability
func (req *Request) GetLoggableStrings() (*RequestLoggableStrings, error) {
	if req == nil {
		return nil, errors.New("cannot read request information from nil request")
	}

	loggableStrings := &RequestLoggableStrings{}
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(req.body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to read request body: err=%+v", err))
	}
	err = req.body.Close()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to close request body ReadCloser: err=%+v", err))
	}

	bodyString := buf.String()
	loggableStrings.Body = bodyString
	req.body = ioutil.NopCloser(strings.NewReader(bodyString))
	return loggableStrings, nil
}
