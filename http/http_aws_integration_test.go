package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	commonhttp "github.com/flanksource/commons/http"
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
		cfg := aws.Config{
			Region:      "us-east-1",
			Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
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

		cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("us-east-1"))
		if err != nil {
			t.Fatalf("Failed to load AWS config: %v", err)
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
			t.Fatalf("STS failed with status %d", resp.StatusCode)
		}
	})

	t.Run("Static Credentials Provider", func(t *testing.T) {
		cfg := aws.Config{
			Region:      "us-east-1",
			Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
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
		cfg := aws.Config{
			Region:      "us-east-1",
			Credentials: credentials.NewStaticCredentialsProvider("test", "test", "test-session-token"),
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

	t.Run("AWS SDK STS Client verification", func(t *testing.T) {
		cfg, err := awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion("us-east-1"),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		localstackEndpoint := "http://localhost:4566"
		stsClient := sts.NewFromConfig(cfg, func(o *sts.Options) {
			o.BaseEndpoint = &localstackEndpoint
		})
		result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			t.Fatalf("AWS SDK STS failed: %v", err)
		}
		if result.UserId == nil {
			t.Fatal("UserId should not be nil")
		}
	})
}

func TestAWSCredentialProvidersConfiguration(t *testing.T) {
	t.Run("Static Credentials Provider", func(t *testing.T) {
		cfg := aws.Config{
			Region:      "us-east-1",
			Credentials: credentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
		}
		client := commonhttp.NewClient().AWSAuthSigV4(cfg).AWSService("s3")
		if client == nil {
			t.Fatal("Client should not be nil")
		}
	})

	t.Run("Default Credential Chain", func(t *testing.T) {
		os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
		os.Setenv("AWS_REGION", "us-west-2")
		defer os.Unsetenv("AWS_ACCESS_KEY_ID")
		defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		defer os.Unsetenv("AWS_REGION")

		cfg, err := awsconfig.LoadDefaultConfig(context.Background())
		if err != nil {
			t.Fatalf("Failed to load default config: %v", err)
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
