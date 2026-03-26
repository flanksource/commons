---
build: echo $GIT_ROOT_DIR==$CWD && pwd &&  cd $GIT_ROOT_DIR && pwd && go build -o hx .
cwd: cmd/hx
exec: $GIT_ROOT_DIR/hx --har "{{ .name | strings.Slug  }}.har"
---

# hx CLI Fixture Tests

## HTTP Methods

| Name | Args | CEL Validation |
|------|------|----------------|
| GET request |  https://httpbin.flanksource.com/get | json.url == "https://httpbin.flanksource.com/get" |
| GET with query params |  https://httpbin.flanksource.com/get search==test page==1 | json.args.search == "test" && json.args.page == "1" |
| POST JSON body |  https://httpbin.flanksource.com/post name=hello count:=42 | json.json.name == "hello" && json.json.count == 42.0 |
| PUT request |  PUT https://httpbin.flanksource.com/put name=updated | json.json.name == "updated" |
| DELETE request |  DELETE https://httpbin.flanksource.com/delete | exitCode == 0 |
| PATCH request |  PATCH https://httpbin.flanksource.com/patch name=patched | json.json.name == "patched" |

## Authentication

| Name | Args | ExitCode | CEL Validation |
|------|------|----------------|----------------|
| Basic auth success |  -u testuser:testpass https://httpbin.flanksource.com/basic-auth/testuser/testpass |  | json.authenticated == true && json.user == "testuser" |
| Basic auth failure |  -u wrong:creds https://httpbin.flanksource.com/basic-auth/testuser/testpass | 1 | stdout.size() == 0 |
| Bearer token |  --token mytoken123 https://httpbin.flanksource.com/bearer |  | json.authenticated == true && json.token == "mytoken123" |
| Digest auth |  --digest -u user:pass https://httpbin.flanksource.com/digest-auth/auth/user/pass |  | json.authenticated == true && json.user == "user" |
| Digest auth wrong password |  --digest -u user:wrong https://httpbin.flanksource.com/digest-auth/auth/user/pass | 1 | exitCode == 1 |

## Status Codes

| Name | ExitCode | Args | CEL Validation |
|------|------|----------------|----------------|
| Status 200 | 0 |  https://httpbin.flanksource.com/status/200 | exitCode == 0 |
| Status 404 | 1 |  https://httpbin.flanksource.com/status/404 | exitCode == 1 |
| Status 500 | 1 |  https://httpbin.flanksource.com/status/500 | exitCode == 1 |

## Headers and Data

| Name | Args | CEL Validation |
|------|------|----------------|
| Custom header |  https://httpbin.flanksource.com/headers X-Custom:hello | json.headers["X-Custom"] == "hello" |
| JSON response |  https://httpbin.flanksource.com/get | stdout.contains('"url"') |
| UUID endpoint |  https://httpbin.flanksource.com/uuid | json.uuid.size() > 0 |
| IP endpoint |  https://httpbin.flanksource.com/ip | stdout.contains('"origin"') |

## OAuth

| Name | Args | CEL Validation |
|------|------|----------------|
| OAuth client_credentials |  --oauth-client-id demo-backend-client --oauth-client-secret MJlO3binatD9jk1 --oauth-token-url https://login-demo.curity.io/oauth/v2/oauth-token --oauth-scope read https://httpbin.flanksource.com/get | json.headers.Authorization.startsWith("Bearer ") |

## AWS SigV4

| Name | Args | CEL Validation |
|------|------|----------------|
| AWS SigV4 signing |  --aws-sigv4 --aws-service execute-api --aws-region us-east-1 https://httpbin.flanksource.com/headers | json.headers.Authorization.startsWith("AWS4-HMAC-SHA256") |

## Miscellaneous

| Name | Args | CEL Validation |
|------|------|----------------|
| Form data POST |  -f key1=val1 -f key2=val2 https://httpbin.flanksource.com/post | json.form.key1 == "val1" && json.form.key2 == "val2" |
