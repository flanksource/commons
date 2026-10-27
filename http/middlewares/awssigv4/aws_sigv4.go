package awssigv4

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AWSSigv4Config struct {
	Region              string
	Service             string
	Endpoint            string
	CredentialsProvider CredentialsProvider
	Tracer              func(msg string)
	// UnsignedPayload leaves streaming bodies unread; S3 accepts UNSIGNED-PAYLOAD.
	UnsignedPayload bool
}

type awsSigv4RoundTripper struct {
	config    AWSSigv4Config
	transport http.RoundTripper
}

func NewAWSSigv4Transport(config AWSSigv4Config, transport http.RoundTripper) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if _, cached := config.CredentialsProvider.(*CredentialsCache); !cached {
		config.CredentialsProvider = NewCredentialsCache(config.CredentialsProvider)
	}

	return &awsSigv4RoundTripper{
		config:    config,
		transport: transport,
	}
}

func (t *awsSigv4RoundTripper) trace(format string, args ...any) {
	if t.config.Tracer != nil {
		t.config.Tracer(fmt.Sprintf(format, args...))
	}
}

func InferServiceFromHost(host string) string {
	// Strip port
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	// e.g. "sts.us-east-1.amazonaws.com" -> "sts"
	// e.g. "s3.amazonaws.com" -> "s3"
	// e.g. "lambda.eu-west-1.amazonaws.com" -> "lambda"
	parts := strings.Split(host, ".")
	if len(parts) >= 3 && strings.HasSuffix(host, ".amazonaws.com") {
		return parts[0]
	}
	return ""
}

func (t *awsSigv4RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())

	service := t.config.Service
	if service == "" {
		service = InferServiceFromHost(req.URL.Host)
	}

	t.trace("aws: signing %s %s for %s/%s", req.Method, req.URL.Host, service, t.config.Region)

	if t.config.Endpoint != "" {
		req.URL.Scheme = "http"
		req.URL.Host = t.config.Endpoint
	}

	credentials, err := t.config.CredentialsProvider.Retrieve(req.Context())
	if err != nil {
		return nil, err
	}

	payloadHash := hashPayload(nil)
	if t.config.UnsignedPayload {
		payloadHash = "UNSIGNED-PAYLOAD"
	} else if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		payloadHash = hashPayload(bodyBytes)
	}
	_, err = Sign(req, SignOptions{Credentials: credentials, Region: t.config.Region, Service: service, Time: time.Now(), PayloadHash: payloadHash})
	if err != nil {
		return nil, err
	}

	return t.transport.RoundTrip(req)
}
