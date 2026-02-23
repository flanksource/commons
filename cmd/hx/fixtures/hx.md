---
build: cd cmd/hx && go build -o fixtures/hx .
exec: ./hx
args: ["--quiet"]
---

# hx CLI Fixture Tests

## HTTP Methods

| Name | Args | CEL Validation |
|------|------|----------------|
| GET request | --quiet https://httpbin.flanksource.com/get | json.url == "https://httpbin.flanksource.com/get" |
| GET with query params | --quiet https://httpbin.flanksource.com/get search==test page==1 | json.args.search == "test" && json.args.page == "1" |
| POST JSON body | --quiet https://httpbin.flanksource.com/post name=hello count:=42 | json.json.name == "hello" && json.json.count == 42.0 |
| PUT request | --quiet PUT https://httpbin.flanksource.com/put name=updated | json.json.name == "updated" |
| DELETE request | --quiet DELETE https://httpbin.flanksource.com/delete | exitCode == 0 |
| PATCH request | --quiet PATCH https://httpbin.flanksource.com/patch name=patched | json.json.name == "patched" |

## Authentication

| Name | Args | CEL Validation |
|------|------|----------------|
| Basic auth success | --quiet -u testuser:testpass https://httpbin.flanksource.com/basic-auth/testuser/testpass | json.authenticated == true && json.user == "testuser" |
| Basic auth failure | --quiet -u wrong:creds https://httpbin.flanksource.com/basic-auth/testuser/testpass | exitCode == 0 && stdout.size() == 0 |
| Bearer token | --quiet --token mytoken123 https://httpbin.flanksource.com/bearer | json.authenticated == true && json.token == "mytoken123" |

## Status Codes

| Name | Args | CEL Validation |
|------|------|----------------|
| Status 200 | --quiet https://httpbin.flanksource.com/status/200 | exitCode == 0 |
| Status 404 | --quiet https://httpbin.flanksource.com/status/404 | exitCode == 0 |
| Status 500 | --quiet https://httpbin.flanksource.com/status/500 | exitCode == 0 |

## Headers and Data

| Name | Args | CEL Validation |
|------|------|----------------|
| Custom header | --quiet https://httpbin.flanksource.com/headers X-Custom:hello | json.headers["X-Custom"] == "hello" |
| JSON response | --quiet https://httpbin.flanksource.com/get | stdout.contains('"url"') |
| UUID endpoint | --quiet https://httpbin.flanksource.com/uuid | json.uuid.size() > 0 |
| IP endpoint | --quiet https://httpbin.flanksource.com/ip | stdout.contains('"origin"') |

## OAuth

| Name | Args | CEL Validation |
|------|------|----------------|
| OAuth client_credentials | --quiet --oauth-client-id demo-backend-client --oauth-client-secret MJlO3binatD9jk1 --oauth-token-url https://login-demo.curity.io/oauth/v2/oauth-token --oauth-scope read https://httpbin.flanksource.com/get | json.headers.Authorization.startsWith("Bearer ") |

## AWS SigV4

| Name | Args | CEL Validation |
|------|------|----------------|
| AWS SigV4 signing | --quiet --aws-sigv4 --aws-service execute-api --aws-region us-east-1 https://httpbin.flanksource.com/headers | json.headers.Authorization.startsWith("AWS4-HMAC-SHA256") |

## Miscellaneous

| Name | Args | CEL Validation |
|------|------|----------------|
| Form data POST | --quiet -f key1=val1 -f key2=val2 https://httpbin.flanksource.com/post | json.form.key1 == "val1" && json.form.key2 == "val2" |
