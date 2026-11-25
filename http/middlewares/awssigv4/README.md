# AWS Signature Version 4

This package signs HTTP requests using the Go standard library. The commons root module has no AWS SDK requirement. The implementation follows the [AWS signing specification](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_sigv-create-signed-request.html); the official AWS vectors are vendored from [botocore revision 2cd57933cf3d0135d7bbfdebcbb8e575af35d1db](https://github.com/boto/botocore/tree/2cd57933cf3d0135d7bbfdebcbb8e575af35d1db/tests/unit/auth/aws4_testsuite), with its Apache-2.0 license in `testdata/LICENSE`.

## Credentials and transport

```go
provider := awssigv4.NewStaticCredentialsProvider(awssigv4.Credentials{
    AccessKeyID: "example-access-key",
    SecretAccessKey: "example-secret-key",
})
client := commonshttp.NewClient().AWSAuthSigV4(awssigv4.Config{
    Region: "us-east-1",
    Credentials: provider,
}).AWSService("s3")
```

`Credentials` contains `AccessKeyID`, `SecretAccessKey`, optional `SessionToken`, and optional `Expires`. Zero `Expires` means that the provider's credentials do not expire. `CredentialsProvider.Retrieve(context.Context)` supplies them. `EnvironmentProvider{}` reads `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN`; the caller supplies the region explicitly.

`NewCredentialsCache(provider)` serializes retrieval and refreshes one minute before expiry, or ten percent of the credential lifetime before expiry for short-lived credentials. Missing keys, expired credentials, cancellation, and provider errors are returned. Refresh failures are surfaced even when a previously cached credential has not yet expired. Both `Client.AWSAuthSigV4` and the transport constructor install the cache without wrapping an existing cache again.

The transport configuration is `AWSSigv4Config`:

| Field | Meaning |
|---|---|
| `Region` | Signing region; required when sending a request |
| `Service` | Service name; otherwise inferred by `InferServiceFromHost` |
| `Endpoint` | Optional host:port override, using HTTP, for local endpoints such as LocalStack |
| `CredentialsProvider` | Commons credentials provider; required |
| `Tracer` | Optional signing-progress callback |
| `UnsignedPayload` | Explicit streaming mode; signs `UNSIGNED-PAYLOAD` without reading or replacing the body |

The default transport buffers and rewinds request bodies, including bodies without `GetBody`, before forwarding them. Body-read errors are returned. Streaming callers can set `UnsignedPayload`; this is an explicit signing policy, never a fallback after a read failure. S3 receives `X-Amz-Content-Sha256`. Other services retain their existing header behavior.

## SDK-free credential providers

Each provider implements `CredentialsProvider`, returns an error rather than empty credentials, and composes with `NewCredentialsCache` for expiry-aware refresh. All take an options struct and use the standard library `net/http` (the commons HTTP client imports this package, so it cannot be used here); pass `HTTPClient` to customise transport, proxies or timeouts.

- `NewAssumeRoleProvider(AssumeRoleOptions)` calls STS `AssumeRole` through the Query API (`POST https://sts.<region>.amazonaws.com/`, `Version=2011-06-15`), signed with SigV4 for service `sts` using the `Base` provider. `Region`, `Base`, `RoleARN` and `SessionName` are required; `ExternalID` and `Duration` (sent as `DurationSeconds`) are optional. `Endpoint` overrides the STS URL, for example a LocalStack address.
- `NewWebIdentityProvider(WebIdentityOptions)` calls the unsigned `AssumeRoleWithWebIdentity`. `TokenFile`, `RoleARN` and `SessionName` default to `AWS_WEB_IDENTITY_TOKEN_FILE`, `AWS_ROLE_ARN` and `AWS_ROLE_SESSION_NAME`. The token file is re-read on every retrieval because projected tokens rotate. When no session name is configured a unique `commons-awssigv4-<nanoseconds>` name is generated.
- `NewIMDSProvider(IMDSOptions)` reads EC2 instance-role credentials with IMDSv2 only: `PUT /latest/api/token`, then `GET /latest/meta-data/iam/security-credentials/` and `/<role>`. There is no IMDSv1 fallback. `Endpoint` defaults to `AWS_EC2_METADATA_SERVICE_ENDPOINT`, then `http://169.254.169.254`; `AWS_EC2_METADATA_DISABLED=true` makes it fail. The default client has a one second connect timeout.
- `Chain(providers...)` returns the first success. When every provider fails, the error is an `errors.Join` of all failures, each labelled with its position and type, so `errors.Is` and `errors.As` still see the individual causes.
- `DefaultChain(region, opts...)` is a cached `Chain` of `EnvironmentProvider`, `WebIdentityProvider` (only when both `AWS_WEB_IDENTITY_TOKEN_FILE` and `AWS_ROLE_ARN` are set) and `IMDSProvider`. Options are `WithHTTPClient`, `WithSTSEndpoint` and `WithIMDSEndpoint`.

STS failures are returned as `*STSError` carrying `StatusCode`, `Code`, `Message` and `RequestID`.

```go
base := awssigv4.DefaultChain("us-east-1")
assumed := awssigv4.NewAssumeRoleProvider(awssigv4.AssumeRoleOptions{
    Region:      "us-east-1",
    Base:        base,
    RoleARN:     "arn:aws:iam::123456789012:role/example",
    SessionName: "example-session",
    Duration:    time.Hour,
})
client := commonshttp.NewClient().AWSAuthSigV4(awssigv4.Config{
    Region:      "us-east-1",
    Credentials: awssigv4.NewCredentialsCache(assumed),
}).AWSService("s3")

var stsErr *awssigv4.STSError
if _, err := assumed.Retrieve(ctx); errors.As(err, &stsErr) {
    log.Printf("STS rejected the request: %s: %s", stsErr.Code, stsErr.Message)
}
```

## Adapt an AWS SDK provider

Applications can keep their SDK credential chain and adapt its result at the boundary. Given `sdkProvider aws.CredentialsProvider`:

```go
provider := awssigv4.CredentialsProviderFunc(func(ctx context.Context) (awssigv4.Credentials, error) {
    value, err := sdkProvider.Retrieve(ctx)
    if err != nil { return awssigv4.Credentials{}, err }
    result := awssigv4.Credentials{AccessKeyID: value.AccessKeyID, SecretAccessKey: value.SecretAccessKey, SessionToken: value.SessionToken}
    if value.CanExpire { result.Expires = value.Expires }
    return result, nil
})
client.AWSAuthSigV4(awssigv4.Config{Region: sdkConfig.Region, Credentials: provider})
```

The AWS SDK stays in the application module. `cmd/hx` uses this adapter to preserve its SDK credential chain. Passing `aws.Config` directly, and calling `AWSAuthSigV4()` without a configuration, are no longer supported.

## Direct signing

`Sign(request, SignOptions)` updates the request's date, session token, authorization, and canonical query. It returns the canonical request, string to sign, and authorization for diagnostics. Supply credentials, region, service, signing time, and the SHA-256 payload hash or `UNSIGNED-PAYLOAD`.

Non-S3 paths are normalized and receive an additional URI-encoding pass. S3 preserves object-key paths and uses a single encoded path. `DisablePathNormalization` suppresses non-S3 path normalization. `DisableURIPathEscaping` uses one AWS encoding pass on the decoded path, as required by the raw-request AWS test-suite convention. `DisableSessionToken` lets a caller add a token after signing when its service requires that policy; tokens normally participate in the signature.

## Verification

Run all 34 request/canonical-request/string-to-sign/authorization vectors, plus cache and body-replay specifications:

```sh
GOWORK=off go test ./http/middlewares/awssigv4 -count=1 -v
```

The separate `sdkcompat` module compares 288 requests with the real SDK signer and asserts identical headers, host, and canonical query. It covers GET/POST, repeated and empty query parameters, reserved query characters, Unicode and escaped paths, S3 and other services, session credentials, and empty/nonempty bodies. The SDK module is excluded from root-module dependency discovery.

The same module cross-checks `AssumeRoleProvider` and `WebIdentityProvider` against the SDK's `sts` client, pointed at a recording server. The form-encoded request bodies (`Action`, `Version`, `RoleArn`, `RoleSessionName`, `ExternalId`, `DurationSeconds`, `WebIdentityToken`) and content type must be identical, the AssumeRole request must be signed with an `sts` credential scope, the web-identity request must be unsigned, and both must parse to the same credentials as the SDK. Signatures themselves are not compared for these calls because the SDK adds its own signed headers.

The test module's `go.work` selects the root checkout, so the cross-check tests the current signer without adding SDK dependencies to the root module. From the repository root, run:

```sh
go -C http/middlewares/awssigv4/sdkcompat test ./... -count=1 -v
```

The LocalStack tests retain their existing short-mode and availability skip conditions. No real AWS account is needed for the vectors or cross-check.
