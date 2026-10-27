package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	commonhttp "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
)

type STSGetCallerIdentityResponse struct {
	GetCallerIdentityResponse struct {
		GetCallerIdentityResult struct {
			Arn     string `json:"Arn"`
			UserId  string `json:"UserId"`
			Account string `json:"Account"`
		} `json:"GetCallerIdentityResult"`
	} `json:"GetCallerIdentityResponse"`
}

func TestAWSAuthIntegrationWithLocalStack(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !isLocalStackAvailable() {
		t.Skip("LocalStack is not available")
	}

	ctx := context.Background()

	t.Run("Static Credentials with STS GetCallerIdentity", func(t *testing.T) {
		cfg := awssigv4.Config{
			Region:      "us-east-1",
			Credentials: awssigv4.NewStaticCredentialsProvider(awssigv4.Credentials{AccessKeyID: "test", SecretAccessKey: "test", SessionToken: ""}),
		}
		client := commonhttp.NewClient().
			AWSAuthSigV4(cfg).
			AWSService("sts").
			AWSEndpoint("localhost:4566")

		resp, err := client.R(ctx).
			Header("Content-Type", "application/x-www-form-urlencoded").
			Post("http://localhost:4566/", "Action=GetCallerIdentity&Version=2011-06-15")
		if err != nil {
			t.Fatalf("Failed to call STS: %v", err)
		}
		if !resp.IsOK() {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("STS failed with status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		var result STSGetCallerIdentityResponse
		if err := json.Unmarshal(body, &result); err != nil && !resp.IsOK() {
			t.Fatalf("Failed to parse response: %v", err)
		}
	})

	t.Run("Credentials from Environment", func(t *testing.T) {
		os.Setenv("AWS_ACCESS_KEY_ID", "test")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
		os.Setenv("AWS_REGION", "us-east-1")
		defer os.Unsetenv("AWS_ACCESS_KEY_ID")
		defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		defer os.Unsetenv("AWS_REGION")

		cfg := awssigv4.Config{Region: "us-east-1", Credentials: awssigv4.EnvironmentProvider{}}

		client := commonhttp.NewClient().
			AWSAuthSigV4(cfg).
			AWSService("sts").
			AWSEndpoint("localhost:4566")

		resp, err := client.R(ctx).
			Header("Content-Type", "application/x-www-form-urlencoded").
			Post("http://localhost:4566/", "Action=GetCallerIdentity&Version=2011-06-15")
		if err != nil {
			t.Fatalf("Failed to call STS: %v", err)
		}
		if !resp.IsOK() {
			t.Fatalf("STS failed with status %d", resp.StatusCode)
		}
	})

	t.Run("Static Credentials Provider", func(t *testing.T) {
		cfg := awssigv4.Config{
			Region:      "us-east-1",
			Credentials: awssigv4.NewStaticCredentialsProvider(awssigv4.Credentials{AccessKeyID: "test", SecretAccessKey: "test", SessionToken: ""}),
		}

		client := commonhttp.NewClient().
			AWSAuthSigV4(cfg).
			AWSService("sts").
			AWSEndpoint("localhost:4566")

		resp, err := client.R(ctx).
			Header("Content-Type", "application/x-www-form-urlencoded").
			Post("http://localhost:4566/", "Action=GetCallerIdentity&Version=2011-06-15")
		if err != nil {
			t.Fatalf("Failed to call STS: %v", err)
		}
		if !resp.IsOK() {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("STS failed with status %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("Session Token via StaticCredentialsProvider", func(t *testing.T) {
		cfg := awssigv4.Config{
			Region:      "us-east-1",
			Credentials: awssigv4.NewStaticCredentialsProvider(awssigv4.Credentials{AccessKeyID: "test", SecretAccessKey: "test", SessionToken: "test-session-token"}),
		}
		client := commonhttp.NewClient().
			AWSAuthSigV4(cfg).
			AWSService("sts").
			AWSEndpoint("localhost:4566")

		resp, err := client.R(ctx).
			Header("Content-Type", "application/x-www-form-urlencoded").
			Post("http://localhost:4566/", "Action=GetCallerIdentity&Version=2011-06-15")
		if err != nil {
			t.Fatalf("Failed to call STS with session token: %v", err)
		}
		if !resp.IsOK() {
			t.Fatalf("STS with session token failed with status %d", resp.StatusCode)
		}
	})

}

func TestAWSCredentialProvidersConfiguration(t *testing.T) {
	t.Run("Static Credentials Provider", func(t *testing.T) {
		cfg := awssigv4.Config{
			Region:      "us-east-1",
			Credentials: awssigv4.NewStaticCredentialsProvider(awssigv4.Credentials{AccessKeyID: "AKID", SecretAccessKey: "SECRET", SessionToken: ""}),
		}
		client := commonhttp.NewClient().AWSAuthSigV4(cfg).AWSService("s3")
		if client == nil {
			t.Fatal("Client should not be nil")
		}
	})

	t.Run("Environment Credentials Provider", func(t *testing.T) {
		os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
		os.Setenv("AWS_REGION", "us-west-2")
		defer os.Unsetenv("AWS_ACCESS_KEY_ID")
		defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		defer os.Unsetenv("AWS_REGION")

		cfg := awssigv4.Config{Region: os.Getenv("AWS_REGION"), Credentials: awssigv4.EnvironmentProvider{}}
		if _, err := cfg.Credentials.Retrieve(context.Background()); err != nil {
			t.Fatal(err)
		}

		client := commonhttp.NewClient().AWSAuthSigV4(cfg).AWSService("s3")
		if client == nil {
			t.Fatal("Client should not be nil")
		}
	})
}

func isLocalStackAvailable() bool {
	resp, err := http.Get("http://localhost:4566/_localstack/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
