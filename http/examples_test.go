package http_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/logger"
)

// TODO: This will be removed or we can keep it as exaples as well
// Contains some examples
func TestExample(t *testing.T) {
	client := http.NewClient().
		SetBaseURL("https://dummyjson.com").
		SetBasicAuth("username", "password").
		SetHost("dummyjson.com").
		SetHeader("Name", "Aditya")

	{
		body := &bytes.Buffer{}
		body.WriteString(`{"title": "test"}`)
		postReq := client.R().SetContext(context.TODO()).SetBody(body).SetHeader("Scope", "request")
		response, err := postReq.Post("products/add")
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
		req := client.R().SetContext(context.TODO())
		response, err := req.Get("products/1")
		if err != nil {
			logger.Fatalf("error: %v", err)
		}

		b, _ := io.ReadAll(response.Body)
		logger.Infof("GET body: %s %v", string(b), response.IsOK())
	}

	{
		// To use tracing
		// tracedTransport := transports.NewTracedTransport(otel.GetTracerProvider().Tracer("http-client")).
		// 	Mode(transports.TraceResponse | transports.TraceBody)

		// client := http.NewClient().WrapTransport(tracedTransport)

		// req := client.R().SetContext(context.TODO())
		// response, err := req.Get("products/1")
		// if err != nil {
		// 	logger.Fatalf("error: %v", err)
		// }

		// b, _ := io.ReadAll(response.Body)
		// logger.Infof("GET body: %s %v", string(b), response.IsOK())
	}
}
