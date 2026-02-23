package http_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	commonhttp "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
)

func TestAWSAuthSigV4(t *testing.T) {
	t.Run("with explicit config", func(t *testing.T) {
		cfg := aws.Config{
			Region:      "us-east-1",
			Credentials: credentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
		}
		client := commonhttp.NewClient().AWSAuthSigV4(cfg).AWSService("s3")
		if client == nil {
			t.Fatal("client should not be nil")
		}
	})

	t.Run("with service override", func(t *testing.T) {
		cfg := aws.Config{
			Region:      "us-west-2",
			Credentials: credentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
		}
		services := []string{"s3", "execute-api", "es", "lambda", "sts"}
		for _, svc := range services {
			t.Run(svc, func(t *testing.T) {
				client := commonhttp.NewClient().AWSAuthSigV4(cfg).AWSService(svc)
				if client == nil {
					t.Fatalf("client should not be nil for service %s", svc)
				}
			})
		}
	})

	t.Run("with custom endpoint", func(t *testing.T) {
		cfg := aws.Config{
			Region:      "us-east-1",
			Credentials: credentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
		}
		client := commonhttp.NewClient().
			AWSAuthSigV4(cfg).
			AWSService("s3").
			AWSEndpoint("localhost:4566")
		if client == nil {
			t.Fatal("client should not be nil")
		}
	})

	t.Run("chaining with other options", func(t *testing.T) {
		cfg := aws.Config{
			Region:      "us-east-1",
			Credentials: credentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
		}
		client := commonhttp.NewClient().
			BaseURL("https://api.example.com").
			Header("X-Custom", "value").
			AWSAuthSigV4(cfg).
			AWSService("execute-api").
			Retry(3, 1, 2.0)
		if client == nil {
			t.Fatal("client should not be nil")
		}
	})
}

func TestAuthConfigIsEmpty(t *testing.T) {
	t.Run("with AWS credentials", func(t *testing.T) {
		config := &commonhttp.AuthConfig{
			AWSCredentialsProvider: credentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
			AWSRegion:              "us-east-1",
		}
		if config.IsEmpty() {
			t.Error("AuthConfig with AWS credentials should not be empty")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		config := &commonhttp.AuthConfig{}
		if !config.IsEmpty() {
			t.Error("empty AuthConfig should be empty")
		}
	})

	t.Run("basic auth only", func(t *testing.T) {
		config := &commonhttp.AuthConfig{Username: "user", Password: "pass"}
		if config.IsEmpty() {
			t.Error("AuthConfig with basic auth should not be empty")
		}
	})
}

func TestInferServiceFromHost(t *testing.T) {
	tests := []struct {
		host    string
		service string
	}{
		{"sts.us-east-1.amazonaws.com", "sts"},
		{"s3.amazonaws.com", "s3"},
		{"s3.us-west-2.amazonaws.com", "s3"},
		{"lambda.eu-west-1.amazonaws.com", "lambda"},
		{"execute-api.us-east-1.amazonaws.com", "execute-api"},
		{"es.us-east-1.amazonaws.com", "es"},
		{"localhost:4566", ""},
		{"example.com", ""},
		{"sts.us-east-1.amazonaws.com:443", "sts"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := middlewares.InferServiceFromHost(tt.host)
			if got != tt.service {
				t.Errorf("InferServiceFromHost(%q) = %q, want %q", tt.host, got, tt.service)
			}
		})
	}
}
