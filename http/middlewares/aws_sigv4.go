package middlewares

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/flanksource/commons/logger"
)

type AWSSigv4Config struct {
	Region              string
	Service             string
	Endpoint            string
	CredentialsProvider aws.CredentialsProvider
	Tracer              func(msg string)
}

type awsSigv4RoundTripper struct {
	config    AWSSigv4Config
	signer    *v4.Signer
	transport http.RoundTripper
}

func NewAWSSigv4Transport(config AWSSigv4Config, transport http.RoundTripper) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}

	return &awsSigv4RoundTripper{
		config:    config,
		signer:    v4.NewSigner(),
		transport: transport,
	}
}

func (t *awsSigv4RoundTripper) trace(format string, args ...any) {
	logger.V(logger.Trace4).Infof(format, args...)
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

	var payloadHash string
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		hash := sha256.Sum256(bodyBytes)
		payloadHash = hex.EncodeToString(hash[:])
	} else {
		hash := sha256.Sum256([]byte{})
		payloadHash = hex.EncodeToString(hash[:])
	}

	err = t.signer.SignHTTP(
		req.Context(),
		credentials,
		req,
		payloadHash,
		service,
		t.config.Region,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	return t.transport.RoundTrip(req)
}
