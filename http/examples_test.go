package http_test

import (
	"bytes"
	"context"
	"io"
	netHTTP "net/http"
	"testing"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/logger"
)

func TestExample(t *testing.T) {
	ctx := context.Background()

	{
		// Skip SSL verification
		req := http.NewClient().InsecureSkipVerify(true).R(ctx)
		response, err := req.Get("https://expired.badssl.com/")
		if err != nil {
			t.Errorf("error: %v", err)
		}

		logger.Infof("GET body: %v", response.IsOK())
	}

	{
		// Use a proxy
		// req := http.NewClient().Timeout(time.Second * 5).Proxy("http://my-proxy.local:1337").R(ctx)
		// response, err := req.Get("https://flanksource.com/")
		// if err != nil {
		// 	logger.Fatalf("error: %v", err)
		// }

		// logger.Infof("GET body: %v", response.IsOK())
	}

	client := http.NewClient().
		BaseURL("https://dummyjson.com").
		BasicAuth("username", "password").
		ConnectTo("dummyjson.com").
		Use(loggerMiddlware).
		Retry(2, time.Second, 2.0).
		Header("API-KEY", "123456")

	{
		body := &bytes.Buffer{}
		body.WriteString(`{"title": "test"}`)
		postReq := client.R(ctx).Header("Scope", "request")
		response, err := postReq.Post("products/add", body)
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
		req := client.R(ctx)
		response, err := req.Get("products/1")
		if err != nil {
			logger.Fatalf("error: %v", err)
		}

		b, _ := io.ReadAll(response.Body)
		logger.Infof("GET body: %s %v", string(b), response.IsOK())
	}

	{
		// To use tracing
		tracedTransport := middlewares.NewTracedTransport().TraceBody(true).TraceResponse(true)

		client := http.NewClient().Use(loggerMiddlware, tracedTransport.RoundTripper)

		req := client.R(ctx)
		response, err := req.Get("https://flanksource.com")
		if err != nil {
			logger.Fatalf("error: %v", err)
		}

		logger.Infof("Status OK: %v", response.IsOK())
	}

	{
		// To use pretty logger
		// tracedTransport := transports.NewTracedTransport().TraceBody(true).TraceResponse(true)

		// httPrettyLogger := &httpretty.Logger{
		// 	Time:           true,
		// 	TLS:            true,
		// 	RequestHeader:  true,
		// 	RequestBody:    true,
		// 	ResponseHeader: true,
		// 	ResponseBody:   true,
		// 	Colors:         true,
		// 	Formatters:     []httpretty.Formatter{&httpretty.JSONFormatter{}},
		// }

		// client := http.NewClient().Use(loggerMiddlware, tracedTransport.RoundTripper, httPrettyLogger.RoundTripper)

		// req := client.R(ctx)
		// response, err := req.Get("https://flanksource.com")
		// if err != nil {
		// 	logger.Fatalf("error: %v", err)
		// }

		// logger.Infof("Status OK: %v", response.IsOK())
	}
}

func loggerMiddlware(next netHTTP.RoundTripper) netHTTP.RoundTripper {
	x := func(req *netHTTP.Request) (*netHTTP.Response, error) {
		logger.Infof("request: %v", req.URL.String())
		return next.RoundTrip(req)
	}

	return http.RoundTripperFunc(x)
}
