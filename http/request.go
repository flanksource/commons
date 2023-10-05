package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/flanksource/commons/dns"
	"github.com/flanksource/commons/logger"
	"go.opentelemetry.io/otel/attribute"
)

type Request struct {
	config *Config
	verb   string
	url    *url.URL
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
func NewGetRequest(config *Config, endpoint string) *GetRequest {
	parsedURL, _ := url.Parse(endpoint)
	return &GetRequest{
		&Request{
			verb:   http.MethodGet,
			url:    parsedURL,
			config: config,
		},
	}
}

// NewPostRequest creates a new HTTP POST request
func NewPostRequest(config *Config, endpoint string, contentType string, body io.ReadCloser) *PostRequest {
	parsedURL, _ := url.Parse(endpoint)
	return &PostRequest{
		&Request{
			verb:   http.MethodPost,
			url:    parsedURL,
			body:   body,
			config: config,
		},
		&ContentType{
			contentType: contentType,
		},
	}
}

// NewPutRequest creates a new HTTP PUT request
func NewPutRequest(config *Config, endpoint string, body io.ReadCloser) *PutRequest {
	parsedURL, _ := url.Parse(endpoint)
	return &PutRequest{
		&Request{
			verb:   http.MethodPut,
			url:    parsedURL,
			body:   body,
			config: config,
		},
	}
}

// NewPatchRequest creates a new HTTP PATCH request
func NewPatchRequest(config *Config, endpoint string, body io.ReadCloser) *PatchRequest {
	parsedURL, _ := url.Parse(endpoint)
	return &PatchRequest{
		&Request{
			verb:   http.MethodPatch,
			url:    parsedURL,
			body:   body,
			config: config,
		},
	}
}

// NewDeleteRequest creates a new HTTP DELETE request
func NewDeleteRequest(config *Config, endpoint string) *DeleteRequest {
	parsedURL, _ := url.Parse(endpoint)
	return &DeleteRequest{
		&Request{
			verb:   http.MethodDelete,
			url:    parsedURL,
			config: config,
		},
	}
}

// Send the HTTP GET request
func (r *GetRequest) Send(ctx context.Context, client *http.Client, logger logger.Logger) (*Response, error) {
	// GET requests are idempotent so can have retries
	var retries uint
	if r.config.Retries != nil {
		retries = r.config.Retries.Total
	}

	return r.sendRequest(ctx, client, logger, retries)
}

// Send the HTTP POST request
func (r *PostRequest) Send(ctx context.Context, client *http.Client, logger logger.Logger) (*Response, error) {
	r.config.Headers[contentType] = r.contentType

	// POST is non-idempotent so can have no retries
	var retries uint = 0
	return r.sendRequest(ctx, client, logger, retries)
}

// Send the HTTP PUT request
func (r *PutRequest) Send(ctx context.Context, client *http.Client, logger logger.Logger) (*Response, error) {
	// PUT is non-idempotent so can have no retries
	var retries uint = 0
	return r.sendRequest(ctx, client, logger, retries)
}

// Send the HTTP PATCH request
func (r *PatchRequest) Send(ctx context.Context, client *http.Client, logger logger.Logger) (*Response, error) {
	// PATCH is non-idempotent so can have no retries
	var retries uint = 0
	return r.sendRequest(ctx, client, logger, retries)
}

// Send the HTTP DELETE request
func (r *DeleteRequest) Send(ctx context.Context, client *http.Client, logger logger.Logger) (*Response, error) {
	// DELETE is non-idempotent so can have no retries
	var retries uint = 0
	return r.sendRequest(ctx, client, logger, retries)
}

// createHTTPRequest configures an HTTP request with the configured values
func (r *Request) createHTTPRequest(ctx context.Context) (*http.Request, error) {
	requestURL := r.url.String()
	if baseURL := strings.TrimSpace(r.config.BaseURL); baseURL != "" {
		requestURL = fmt.Sprintf("%s/%s", baseURL, r.url.String())
	}

	request, err := http.NewRequestWithContext(ctx, r.verb, requestURL, r.body)
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
func (r *Request) sendRequest(ctx context.Context, client *http.Client, logger logger.Logger, retriesRemaining uint) (*Response, error) {
	ctx, span := tracer.Start(ctx, r.url.Hostname()) // TODO: Decide on how to name this span
	defer span.End()

	span.SetAttributes(attribute.String("method", r.verb))

	if r.config.ConnectTo == "" {
		r.config.ConnectTo = r.url.Hostname()
	} else if r.config.ConnectTo != r.url.Hostname() {
		// If specified, replace the hostname in the URL, with the actual host/IP
		// and move the Virtual Hostname to a Header
		r.url.Host = r.config.ConnectTo
	}

	if r.config.Headers["Host"] != "" {
		r.config.ConnectTo = r.url.Hostname()
		r.url.Host = r.config.Headers["Host"]
		port := r.url.Port()
		if port != "" {
			r.url.Host += ":" + port
		}

		delete(r.config.Headers, "Host")
	}

	if r.config.ConnectTo == "" && r.config.DNSCache {
		ips, err := dns.CacheLookup(ctx, "A", r.url.Hostname())
		if len(ips) == 0 {
			return nil, err
		}

		r.config.ConnectTo = ips[0].String()
	}

	request, err := r.createHTTPRequest(ctx)
	if Error(span, err) {
		return nil, err
	}

	response, err := client.Do(request)
	if Error(span, err) {
		// if the retries have been exhausted (or not configured), bail out
		if retriesRemaining <= 0 {
			return nil, err
		}

		retriesRemaining--

		if r.config.Retries != nil {
			backoffTime := exponentialBackoff(r.config.Retries, retriesRemaining)
			logger.Warnf("backing off for %v before next retry", backoffTime)
		}

		return r.sendRequest(ctx, client, logger, retriesRemaining)
	}

	if r.config.TraceBody {
		body, err := io.ReadAll(response.Body)
		if Error(span, err) {
			return nil, err
		}

		response.Body = io.NopCloser(bytes.NewBuffer(body))
		span.SetAttributes(attribute.String("body", string(body)))
	}

	res := Response(*response)
	return &res, nil
}

type RequestLoggableStrings struct {
	Headers string
	Body    string
}

// GetLoggableStrings returns the Headers and Body of the response as strings that can be logged while
// maintaining the request body's readability
func (r *Request) GetLoggableStrings() (string, error) {
	if r == nil {
		return "", errors.New("cannot read request information from nil request")
	}

	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.body)
	if err != nil {
		return "", fmt.Errorf("failed to read request body: err=%+v", err)
	}

	err = r.body.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close request body ReadCloser: err=%+v", err)
	}

	bodyString := buf
	r.body = io.NopCloser(bufio.NewReader(buf))

	return fmt.Sprintf("body=<%s>", bodyString), nil
}
