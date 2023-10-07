package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// Response extends the stdlib http.Response type and extends its functionality
type Response struct {
	// The underlying http.Response is embed into Response.
	*http.Response

	// Request is the Response's related Request.
	Request *Request

	Err error
}

// IsOK is a convenience method to determine if the response returned a 200 OK
func (resp *Response) IsOK(responseCodes ...int) bool {
	if len(responseCodes) == 0 {
		return resp.StatusCode >= 200 && resp.StatusCode < 299
	}

	for _, valid := range responseCodes {
		if resp.StatusCode == valid {
			return true
		}
	}

	return false
}

func (r *Response) Into(dest any) error {
	if r.Err != nil {
		return r.Err
	}

	contentType := r.Header.Get(contentType)
	if strings.Contains(contentType, "json") {
		return json.NewDecoder(r.Body).Decode(dest)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	dest = body //nolint:ineffassign // Cannot enforce dest to be a pointer

	return nil
}

// TraceMessage returns the Headers, StatusCode and Body of the response as strings that can be logged while
// maintaining the response body's readability
func (resp *Response) TraceMessage() (string, error) {
	if resp == nil {
		return "", errors.New("cannot read response information from nil response")
	}

	traceMessage := fmt.Sprintf("status=%d, content-type=<%s>", resp.StatusCode, resp.Header.Get(contentType))
	buf := new(bytes.Buffer)
	_, readErr := buf.ReadFrom(resp.Body)
	if readErr != nil {
		return traceMessage, nil
	}
	defer resp.Body.Close()
	traceMessage += fmt.Sprintf("\n%+v", buf)

	return traceMessage, nil
}
