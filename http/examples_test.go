package http_test

import (
	"bytes"
	"context"
	"io"
	netHTTP "net/http"
	"testing"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/transports"
	"github.com/flanksource/commons/logger"
)

// TODO: This will be removed or we can keep it as exaples as well
// Contains some examples
func TestExample(t *testing.T) {
	ctx := context.Background()

	client := http.NewClient().
		BaseURL("https://dummyjson.com").
		BasicAuth("username", "password").
		Host("dummyjson.com").
		Use(loggerMiddlware).
		Header("API-KEY", "123456")

	{
		body := &bytes.Buffer{}
		body.WriteString(`{"title": "test"}`)
		postReq := client.R().Header("Scope", "request")
		response, err := postReq.Post(ctx, "products/add", body)
		if err != nil {
			logger.Fatalf("error: %v", err)
		}

		var bodyResponse = map[string]any{}
		if err := response.Into(&bodyResponse); err != nil {
			logger.Fatalf("error: %v", err)
		}
		logger.Infof("body: %v %v", bodyResponse, response.IsOK())
	}

	{
		req := client.R()
		response, err := req.Get(ctx, "products/1")
		if err != nil {
			logger.Fatalf("error: %v", err)
		}

		b, _ := io.ReadAll(response.Body)
		logger.Infof("GET body: %s %v", string(b), response.IsOK())
	}

	{
		// To use tracing
		tracedTransport := transports.NewTracedTransport().TraceBody(true).TraceResponse(true)

		client := http.NewClient().Use(loggerMiddlware, tracedTransport.Middleware)

		req := client.R()
		response, err := req.Get(ctx, "https://flanksource.com")
		if err != nil {
			logger.Fatalf("error: %v", err)
		}

		logger.Infof("Status OK: %v", response.IsOK())
	}
}

func loggerMiddlware(next netHTTP.RoundTripper) netHTTP.RoundTripper {
	x := func(req *netHTTP.Request) (*netHTTP.Response, error) {
		logger.Infof("request: %v", req.URL.String())
		return next.RoundTrip(req)
	}

	return http.RoundTripperFunc(x)
}
