package net

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)


// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestGetReturns_Success(t *testing.T) {

	client := NewTestClient(func(req *http.Request) *http.Response {
		// Test request parameters
		assert.Equal(t, req.URL.String(), "http://example.com")
		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString(`OK`)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})

	api := API{client, "http://example.com"}
	body, err := api.GET()
	assert.NoError(t, err)
	assert.Equal(t, []byte("OK"), body)
	assert.NotEmpty(t, body)

}

func TestGetReturns_Failure(t *testing.T) {

	client := NewTestClient(func(req *http.Request) *http.Response {
		// Test request parameters
		assert.Equal(t, req.URL.String(), "http://example.com")
		return &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
			Header:     make(http.Header),
		}
	})

	api := API{client, "http://example.com"}
	body, err := api.GET()
	assert.Error(t, err)
	assert.Empty(t, body)

}

func Test_NewClient(t *testing.T) {

	resp := NewClient("/test.go")

	assert.Equal(t, resp.baseURL, "/test.go")
	assert.IsType(t, resp, &API{})

}

