package http_test

import (
	"context"
	"fmt"
	"net"
	netHTTP "net/http"
	"net/url"
	"testing"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/logger"
)

// Test with few example use cases.
// Disabled because Github action doesn't allow making external calls?
// Responds with 403 & 421
// https://github.com/flanksource/commons/actions/runs/6458930480/job/17533665387?pr=79
// nolint:unused
func TestHTTP(t *testing.T) {
	ctx := context.Background()

	t.Run("OAuth", func(t *testing.T) {
		t.SkipNow()

		var (
			clientID     = ""
			clientSecret = ""
			tenantID     = ""
			tokenURL     = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
			scopes       = []string{"https://graph.microsoft.com/.default"}
		)

		req := http.NewClient().OAuth(
			http.OauthConfig{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				TokenURL:     tokenURL,
				Scopes:       scopes}).
			R(ctx)
		response, err := req.Get("https://graph.microsoft.com/v1.0/users")
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		body, err := response.AsJSON()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		t.Logf("body: %v", body)

		if !response.IsOK() {
			t.Fatalf("Got bad response: %d", response.StatusCode)
		}
	})

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
		req := http.NewClient().CacheDNS(true).TraceToStdout(http.TraceAll).R(ctx)
		for i := 0; i < 5; i++ {
			response, err := req.Get("https://github.com/")
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
			if err := response.Into(&bodyResponse); err != nil {
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

	t.Run("Host Header", func(t *testing.T) {
		uri, _ := url.Parse("https://httpbin.demo.aws.flanksource.com/headers")

		ips, err := net.LookupIP(uri.Host)

		if err != nil {
			t.Error(err.Error())
		}

		uriIP := *uri
		uriIP.Host = ips[0].To4().String()

		resp, err := http.NewClient().
			TraceToStdout(http.TraceAll).
			InsecureSkipVerify(true).
			R(context.Background()).
			Header("Host", uri.Host).
			Get(uriIP.String())
		if err != nil {
			t.Error(err)
		}
		var headers map[string]any
		if body, err := resp.AsJSON(); err != nil {
			t.Error(err)
		} else {
			headers = body["headers"].(map[string]any)
		}
		if headers["Host"] != uri.Host {
			t.Errorf("Expected response headers %s", headers)
		}
	})

	t.Run("No Auth", func(t *testing.T) {
		resp, err := http.NewClient().R(context.Background()).Header("Hello", "World").Get("https://httpbin.demo.aws.flanksource.com/headers")
		if err != nil {
			t.Error(err)
		}
		var headers map[string]any
		if body, err := resp.AsJSON(); err != nil {
			t.Error(err)
		} else {
			headers = body["headers"].(map[string]any)
		}
		if headers["Hello"] != "World" {
			t.Errorf("Expected response headers %s", headers)
		}
		if v, ok := headers["Authorization"]; ok {
			t.Errorf("Expecting blank authentication got %s", v)
		}
	})

	t.Run("Tracing & logging middleware", func(t *testing.T) {
		client := http.NewClient().Trace(http.TraceConfig{
			MaxBodyLength: 4096,
			QueryParam:    true,
			Headers:       true,
		})

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

// nolint:unused
func loggerMiddlware(next netHTTP.RoundTripper) netHTTP.RoundTripper {
	x := func(req *netHTTP.Request) (*netHTTP.Response, error) {
		logger.Infof("request: %v", req.URL.String())
		return next.RoundTrip(req)
	}

	return middlewares.RoundTripperFunc(x)
}
