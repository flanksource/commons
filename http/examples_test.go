package http_test

import (
	"context"
	netHTTP "net/http"
	"testing"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/logger"
)

func TestExample(t *testing.T) {
	ctx := context.Background()

	t.Run("Skip SSL Verification", func(t *testing.T) {
		req := http.NewClient().InsecureSkipVerify(true).R(ctx)
		response, err := req.Get("https://expired.badssl.com/")
		if err != nil {
			t.Errorf("error: %v", err)
		}

		if !response.IsOK() {
			t.Errorf("Got bad response: %d", response.StatusCode)
		}
	})

	t.Run("Cache DNS", func(t *testing.T) {
		req := http.NewClient().CacheDNS(true).R(ctx)
		for i := 0; i < 5; i++ {
			response, err := req.Get("https://flanksource.com")
			if err != nil {
				t.Errorf("error: %v", err)
			}

			if !response.IsOK() {
				t.Errorf("Got bad response: %d", response.StatusCode)
			}
		}
	})

	// t.Run("Use a proxy", func(t *testing.T) {
	// req := http.NewClient().Timeout(time.Second * 5).Proxy("http://my-proxy.local:1337").R(ctx)
	// response, err := req.Get("https://flanksource.com/")
	// if err != nil {
	// 	logger.Fatalf("error: %v", err)
	// }

	// logger.Infof("GET body: %v", response.IsOK())
	// })

	t.Run("example GET & POST with basic logging middleware", func(t *testing.T) {
		client := http.NewClient().
			BaseURL("https://dummyjson.com").
			Auth("username", "password").
			ConnectTo("dummyjson.com").
			Use(loggerMiddlware).
			Retry(2, time.Second, 2.0).
			Header("API-KEY", "123456")

		{
			postReq := client.R(ctx).Header("Scope", "request")
			response, err := postReq.Post("products/add", map[string]string{"title": "test"})
			if err != nil {
				t.Errorf("error: %v", err)
			}

			var bodyResponse = map[string]any{}
			if err := response.AsJSON(&bodyResponse); err != nil {
				t.Errorf("error: %v", err)
			}

			if !response.IsOK() {
				t.Errorf("Got bad response: %d", response.StatusCode)
			}
		}

		{
			req := client.R(ctx)
			response, err := req.Get("products/1")
			if err != nil {
				t.Errorf("error: %v", err)
			}

			if !response.IsOK() {
				t.Errorf("Got bad response: %d", response.StatusCode)
			}
		}
	})

	t.Run("Tracing & logging middleware", func(t *testing.T) {
		tracedTransport := middlewares.NewTracedTransport().TraceBody(true).TraceResponse(true)

		client := http.NewClient().Use(loggerMiddlware, tracedTransport.RoundTripper)

		req := client.R(ctx)
		response, err := req.Get("https://flanksource.com")
		if err != nil {
			t.Errorf("error: %v", err)
		}

		if !response.IsOK() {
			t.Errorf("Got bad response: %d", response.StatusCode)
		}
	})

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
