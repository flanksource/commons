# AWS Sigv4 Integration Tests

This directory contains integration tests for AWS Signature Version 4 authentication using LocalStack.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.23+ installed

## Running Integration Tests

### 1. Start LocalStack

```bash
cd /home/runner/work/commons/commons/http/testdata
docker-compose up -d
```

Wait a few seconds for LocalStack to start, then verify it's running:

```bash
curl http://localhost:4566/_localstack/health
```

### 2. Run Integration Tests

```bash
# Run all AWS integration tests
go test -v -run TestAWSAuthIntegrationWithLocalStack ./http

# Run all AWS tests including unit and integration
go test -v -run TestAWS ./http

# Run credential provider tests
go test -v -run TestAWSCredentialProviders ./http
```

### 3. Stop LocalStack

```bash
cd /home/runner/work/commons/commons/http/testdata
docker-compose down
```

## Test Coverage

The integration tests cover:

1. **Static Credentials**: Direct access key/secret key authentication with STS GetCallerIdentity
2. **Environment Variables**: Credentials loaded from `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`
3. **Static Credentials Provider**: Using AWS SDK's `credentials.NewStaticCredentialsProvider`
4. **Session Token Support**: Temporary credentials with session tokens
5. **Custom Endpoint**: Using LocalStack as a custom endpoint
6. **AWS SDK Verification**: Testing with the official AWS STS SDK to verify compatibility

## Test Method

The integration tests use **AWS STS GetCallerIdentity** API call, which is a simple, read-only operation that:
- Requires no setup or resources
- Works with any AWS credentials
- Returns information about the caller (UserId, Account, Arn)
- Is ideal for testing authentication without side effects

This is more suitable than S3 operations as it:
- Doesn't require bucket creation/cleanup
- Tests pure authentication without resource management
- Is faster and simpler
- Works consistently across all credential types

## Running Tests in CI/CD

To run these tests in CI/CD, you can use the following approach:

```yaml
# Example GitHub Actions workflow
steps:
  - name: Start LocalStack
    run: |
      cd http/testdata
      docker-compose up -d
      sleep 5  # Wait for LocalStack to be ready

  - name: Run Integration Tests
    run: go test -v -run TestAWSAuthIntegrationWithLocalStack ./http

  - name: Stop LocalStack
    if: always()
    run: |
      cd http/testdata
      docker-compose down
```

## Test Structure

- `http_aws_test.go`: Unit tests for AWS auth configuration
- `http_aws_integration_test.go`: Integration tests with LocalStack using STS GetCallerIdentity
- `testdata/docker-compose.yml`: LocalStack configuration

## Troubleshooting

### LocalStack not starting

```bash
# Check LocalStack logs
docker-compose -f http/testdata/docker-compose.yml logs

# Restart LocalStack
docker-compose -f http/testdata/docker-compose.yml restart
```

### Tests failing with connection errors

Ensure LocalStack is running and accessible:

```bash
# Should return JSON with service health status
curl http://localhost:4566/_localstack/health
```

### Port 4566 already in use

```bash
# Find and stop process using port 4566
lsof -i :4566
kill <PID>

# Or use a different port by modifying docker-compose.yml
```
