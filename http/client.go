package http

import (
	"fmt"
	"github.com/flanksource/commons/logger"
	"io"
	"net/http"
	"strings"
)

const contentType = "Content-Type"

var contentTypesToLog = []string{
	"text",
	"json",
	"yml",
}

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
	c.logRequest(request.Request, c.config.Logger.Tracef)
	response, err := request.Send(c.httpClient, c.config.Logger)
	c.logResponse(request.verb, c.config.Logger.Tracef, url, response, err)
	return response, err
}

// Post sends an HTTP POST request
func (c *Client) Post(url string, contentType string, body io.ReadCloser) (*Response, error) {
	request := NewPostRequest(c.config, url, contentType, body)
	c.logRequest(request.Request, c.config.Logger.Debugf)

	response, err := request.Send(c.httpClient, c.config.Logger)
	c.logResponse(request.verb, c.config.Logger.Debugf, url, response, err)
	return response, err
}

// Patch sends an HTTP PATCH request
func (c *Client) Patch(url string, body io.ReadCloser) (*Response, error) {
	request := NewPatchRequest(c.config, url, body)
	c.logRequest(request.Request, c.config.Logger.Debugf)

	response, err := request.Send(c.httpClient, c.config.Logger)
	c.logResponse(request.verb, c.config.Logger.Debugf, url, response, err)
	return response, err
}

// Put sends an HTTP PUT request
func (c *Client) Put(url string, body io.ReadCloser) (*Response, error) {
	request := NewPutRequest(c.config, url, body)
	c.logRequest(request.Request, c.config.Logger.Debugf)

	response, err := request.Send(c.httpClient, c.config.Logger)
	c.logResponse(request.verb, c.config.Logger.Debugf, url, response, err)
	return response, err
}

// Delete sends an HTTP DELETE request
func (c *Client) Delete(url string) (*Response, error) {
	request := NewDeleteRequest(c.config, url)
	c.logRequest(request.Request, c.config.Logger.Infof)

	response, err := request.Send(c.httpClient, c.config.Logger)
	c.logResponse(request.verb, c.config.Logger.Infof, url, response, err)
	return response, err
}

func (c *Client) logRequest(request *Request, logFunc func(message string, args ...interface{})) {
	if !c.config.Trace {
		return
	}

	if request == nil {
		c.config.Logger.Tracef("Empty request. Nothing to log: request=%s", request)
		return
	}

	message := fmt.Sprintf("HTTP %s Request: url=%s", request.verb, request.url)
	if !c.config.TraceBody {
		logFunc(message)
		return
	}

	var bodyContentType string
	if c.config.Headers != nil {
		bodyContentType = c.config.Headers[contentType]
	}

	if !shouldLogBody(bodyContentType) {
		message += fmt.Sprintf(", content-type='%s', err='Not logging body: content-type not loggable'", bodyContentType)
		logFunc(message)
		return
	}

	if request.body == nil {
		logFunc(message)
		return
	}

	loggableStrings, err := request.GetLoggableStrings()
	if err != nil {
		message += fmt.Sprintf(", err=%+v", err)
		logFunc(message)
		return
	}
	message += fmt.Sprintf(", content-type='%s', body='%s'", bodyContentType, loggableStrings.Body)
	logFunc(message)
}

func (c *Client) logResponse(verb string, logFunc func(message string, args ...interface{}), url string, response *Response, err error) {
	if !c.config.Trace {
		return
	}

	if !c.config.TraceResponse {
		return
	}

	message := fmt.Sprintf("HTTP %s Response: url=%s", verb, url)
	if response == nil {
		logFunc(message)
		return
	}

	if !c.config.TraceBody {
		logFunc(message)
		return
	}

	bodyContentType := response.Header[contentType]
	var bodyContentTypeString string
	if len(bodyContentType) > 0 {
		bodyContentTypeString = bodyContentType[0]
	}

	if !shouldLogBody(bodyContentTypeString) {
		message += fmt.Sprintf(", content-type='%s', err='Not logging body: content-type not loggable'", bodyContentTypeString)
		logFunc(message)
		return
	}

	loggableStrings := &ResponseLoggableStrings{}
	loggableStrings, err = response.GetLoggableStrings()
	if err != nil {
		message += fmt.Sprintf("content-type='%s', err=%+v", bodyContentTypeString, err)
		logFunc(message)
		return
	}

	message += fmt.Sprintf(", status=%s, content-type='%s', body='%s'",
		loggableStrings.StatusCode, bodyContentTypeString, loggableStrings.Body)
	logFunc(message)
}

func isContentTypeLoggable(contentType string) bool {
	for _, contentTypeToLog := range contentTypesToLog {
		if strings.Contains(contentType, contentTypeToLog) {
			return true
		}
	}
	return false
}

func shouldLogBody(bodyContentType string) bool {
	if bodyContentType == "" {
		return false
	}
	if len(bodyContentType) < 1 {
		return false
	}
	return isContentTypeLoggable(bodyContentType)
}
