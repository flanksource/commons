package http

import (
	"encoding/json"
	"io"
	"net/http"
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

func (r *Response) AsJSON(dest any) error {
	return json.NewDecoder(r.Response.Body).Decode(dest)
}

func (r *Response) AsString() (string, error) {
	res, err := io.ReadAll(r.Response.Body)
	if err != nil {
		return "", err
	}
	defer r.Response.Body.Close()

	return string(res), nil
}
