package sdkcompat_test

import (
	"context"
	"net/http"
	"testing"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func TestAWSAuthSDKIntegrationWithLocalStack(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	response, err := http.Get("http://localhost:4566/_localstack/health")
	if err != nil {
		t.Skip("LocalStack is not available")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Skip("LocalStack is not available")
	}
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("us-east-1"), awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")))
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "http://localhost:4566"
	client := sts.NewFromConfig(cfg, func(options *sts.Options) { options.BaseEndpoint = &endpoint })
	result, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		t.Fatalf("AWS SDK STS failed: %v", err)
	}
	if result.UserId == nil {
		t.Fatal("UserId should not be nil")
	}
}
