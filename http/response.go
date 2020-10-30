package http

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// Response embeds the stdlib http.Response type and extends its functionality
type Response struct {
	*http.Response
}

// IsOK is a convenience method to determine if the response returned a 200 OK
func (resp *Response) IsOK() bool {
	if resp == nil {
		return false
	}

	return resp.StatusCode == http.StatusOK
}

// AsError returns an error with details of the response
func (resp *Response) AsError() error {
	if resp == nil {
		return errors.New("http client did not return a response")
	}

	return errors.Errorf("http client received error response: %v", resp.StatusCode)
}

// AsString returns the body of the response as a string, or returns an error if this is not possible
func (resp *Response) AsString() (string, error) {
	if resp == nil {
		return "", errors.New("cannot read body from nil response")
	}

	body, err := resp.AsBytes()
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// AsReader returns the response body as an io.Reader, or returns an error if this is not possible
func (resp *Response) AsReader() (io.Reader, error) {
	if resp == nil {
		return nil, errors.New("cannot return reader from nil response")
	}

	return resp.Body, nil
}

// AsBytes returns the body of the response as a byte slice, or returns an error if this is not possible
func (resp *Response) AsBytes() ([]byte, error) {
	if resp == nil {
		return nil, errors.New("cannot read body from nil response")
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return nil, errors.Wrap(err, "cannot read response body")
	}

	return body, nil
}

type ResponseLoggableStrings struct {
	Headers    string
	StatusCode string
	Body       string
}

// GetLoggableStrings returns the Headers, StatusCode and Body of the response as strings that can be logged while
// maintaining the response body's readability
func (resp *Response) GetLoggableStrings() (*ResponseLoggableStrings, error) {
	if resp == nil {
		return nil, errors.New("cannot read response information from nil response")
	}

	loggableStrings := new(ResponseLoggableStrings)
	loggableStrings.StatusCode = fmt.Sprintf("%d", resp.StatusCode)

	buf := new(bytes.Buffer)
	_, readErr := buf.ReadFrom(resp.Body)
	readErr = resp.Body.Close()
	if readErr == nil {
		loggableStrings.Body = buf.String()
		resp.Body = ioutil.NopCloser(strings.NewReader(loggableStrings.Body))
	}
	loggableStrings.Headers = fmt.Sprintf("%+v", resp.Header)
	return loggableStrings, nil
}
