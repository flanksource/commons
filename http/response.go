package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// Response extends the stdlib http.Response type and extends its functionality
type Response struct {
	// The underlying http.Response is embed into Response.
	*http.Response

	// Request is the Response's related Request.
	Request *Request
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
	return json.NewDecoder(r.Response.Body).Decode(dest)
}

func (h *Response) AsJSON() (map[string]any, error) {
	var result map[string]any
	if err := h.Into(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Response) AsString() (string, error) {
	res, err := io.ReadAll(r.Response.Body)
	if err != nil {
		return "", err
	}
	defer r.Response.Body.Close()

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
