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

Create a test workspace once, from the repository root:

```sh
mkdir -p .tmp/sdkcompat-work
GOTOOLCHAIN=go1.26.1 GOWORK=off go -C .tmp/sdkcompat-work work init ../.. ../../http/middlewares/awssigv4/sdkcompat
```

Then run the cross-check:

```sh
GOWORK="$PWD/.tmp/sdkcompat-work/go.work" go test ./http/middlewares/awssigv4/sdkcompat -count=1 -v
```

The LocalStack tests retain their existing short-mode and availability skip conditions. No real AWS account is needed for the vectors or cross-check.
