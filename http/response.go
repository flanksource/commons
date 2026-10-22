package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/commons/logger"
)

// ErrBodyConsumed is returned when a response body is read after an earlier
// read (Into, AsJSON, AsString or a direct Body read) already consumed it.
// Enable Client.BufferBody to read a body more than once.
var ErrBodyConsumed = errors.New("response body already consumed; enable Client.BufferBody to read it more than once")

// DefaultMaxBufferedBodyBytes is the response size Client.BufferBody(0) buffers.
const DefaultMaxBufferedBodyBytes int64 = 16 << 20

// Response extends the stdlib http.Response type and extends its functionality
type Response struct {
	// The underlying http.Response is embed into Response.
	*http.Response

	// Request is the Response's related Request.
	Request    *Request
	RawRequest *http.Request

	// buffered holds the whole body when the client enabled BufferBody, so
	// every reader sees it; nil means the body streams and can be read once.
	buffered []byte
}

// trackedBody records whether anything has read the stream yet, so a second
// read fails with ErrBodyConsumed instead of returning an empty body.
type trackedBody struct {
	io.ReadCloser
	read bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	b.read = true
	return b.ReadCloser.Read(p)
}

// newResponse wraps httpResponse, buffering its body (up to maxBuffered bytes)
// when maxBuffered > 0, else tracking reads of the stream.
func newResponse(httpResponse *http.Response, maxBuffered int64) (*Response, error) {
	response := &Response{Response: httpResponse}
	if httpResponse.Body == nil {
		return response, nil
	}
	if maxBuffered <= 0 {
		httpResponse.Body = &trackedBody{ReadCloser: httpResponse.Body}
		return response, nil
	}
	defer func() { _ = httpResponse.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxBuffered+1))
	if err != nil {
		return nil, fmt.Errorf("%s: reading response body: %w", requestLabel(httpResponse.Request), err)
	}
	if int64(len(body)) > maxBuffered {
		return nil, fmt.Errorf("%s: response body exceeds %d byte buffer limit", requestLabel(httpResponse.Request), maxBuffered)
	}
	response.buffered = body
	httpResponse.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}

func requestLabel(req *http.Request) string {
	if req == nil || req.URL == nil {
		return "response"
	}
	return req.Method + " " + req.URL.String()
}

// bodyReader returns a reader over the whole body, or ErrBodyConsumed when an
// unbuffered body has already been read.
func (r *Response) bodyReader() (io.Reader, error) {
	if r.buffered != nil {
		return bytes.NewReader(r.buffered), nil
	}
	if r.Response == nil || r.Response.Body == nil {
		return nil, fmt.Errorf("response has no body")
	}
	if tracked, ok := r.Response.Body.(*trackedBody); ok && tracked.read {
		return nil, fmt.Errorf("%s: %w", requestLabel(r.Response.Request), ErrBodyConsumed)
	}
	return r.Response.Body, nil
}

func (r *Response) GetHeaders() map[string]string {
	return toMap(r.Header)
}

// IsOK is a convenience method to determine if the response returned a 200 OK
func (r *Response) IsOK(responseCodes ...int) bool {
	if len(responseCodes) == 0 {
		return r.StatusCode >= 200 && r.StatusCode < 299
	}

	for _, valid := range responseCodes {
		if r.StatusCode == valid {
			return true
		}
	}

	return false
}

func (r *Response) Into(dest any) error {
	body, err := r.bodyReader()
	if err != nil {
		return err
	}
	return json.NewDecoder(body).Decode(dest)
}

func (h *Response) AsJSON() (map[string]any, error) {
	var result map[string]any
	if err := h.Into(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Response) AsString() (string, error) {
	if r.buffered != nil {
		return string(r.buffered), nil
	}
	body, err := r.bodyReader()
	if err != nil {
		return "", err
	}
	defer r.Response.Body.Close()
	res, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func (h *Response) GetSSLAge() *time.Duration {
	if h.Response == nil || h.Response.TLS == nil {
		return nil
	}

	certificates := h.Response.TLS.PeerCertificates
	if len(certificates) == 0 {
		return nil
	}

	age := time.Until(certificates[0].NotAfter)
	return &age
}

func (h *Response) IsJSON() bool {
	contentType := h.Header["Content-Type"]
	if len(contentType) == 0 {
		return false
	}

	for _, ct := range contentType {
		if strings.Contains(strings.ToLower(ct), "application/json") {
			return true
		}
	}

	return false
}

func (h *Response) HeaderMap() map[string]string {
	headers := make(map[string]string)
	for k, v := range h.Header {
		headers[k] = strings.Join(v, ", ")
	}
	return headers
}

func (h *Response) AsMap() (map[string]any, error) {

	m := make(map[string]any)

	m["headers"] = map[string]any{}
	for k, v := range logger.StripSecretsFromMap(h.HeaderMap()) {
		m["headers"].(map[string]any)[k] = v
	}

	if h.Request != nil && h.Request.url != nil {
		m["url"] = h.Request.url.String()
	}
	m["status"] = h.StatusCode

	var err error
	if h.IsJSON() {
		m["body"], err = h.AsJSON()
	} else {
		m["body"], err = h.AsString()
	}
	if errors.Is(err, ErrBodyConsumed) {
		return nil, err
	}

	return m, nil
}

func (h *Response) Debug() string {
	if h.Response == nil {
		return "nil response"
	}
	// mimic the response, + add content-type and size
	var sb strings.Builder

	if h.Request != nil {
		sb.WriteString(h.Request.Debug())
	}

	fmt.Fprintf(&sb, "\n====> Status: %d\n", h.StatusCode)
	for k, v := range logger.StripSecretsFromMap(h.HeaderMap()) {
		fmt.Fprintf(&sb, "  %s: %s\n", console.Grayf("%s", k), v)
	}
	if _, err := h.bodyReader(); errors.Is(err, ErrBodyConsumed) {
		sb.WriteString("<body already consumed>")
	} else if h.IsJSON() {
		r, _ := h.AsJSON()
		sb.WriteString(logger.Pretty(r))
	} else {
		body, _ := h.AsString()
		sb.WriteString(body)
	}
	return sb.String()
}
