# HTTP client Server-Timing

The fluent client automatically records outgoing HTTP work when its request context carries a Clicky timing collector. Calls without a collector retain their normal behavior. Install the collector at the server boundary, before authentication and other middleware that makes outgoing calls.

```go
handler := rpchttp.TimingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    response, err := httpclient.NewClient().R(r.Context()).Get("https://example.com/status")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadGateway)
        return
    }
    defer response.Body.Close()
    if _, err := io.Copy(w, response.Body); err != nil {
        return
    }
}))
```

Import `rpchttp` from `github.com/flanksource/clicky/rpc/http` and `httpclient` from `github.com/flanksource/commons/http`. The example also uses the standard `net/http` and `io` packages. `R` uses the incoming request's context, preserving cancellation and its collector.

`http` measures each transport exchange until response headers arrive, counting requests, transport errors and HTTP status codes of 400 or higher. Redirect hops and retry attempts are separate exchanges. `http_body` measures the interval from headers until EOF, a read error or close; it counts completed bodies, consumed bytes and body errors. It does not buffer or eagerly consume the stream. A body left open cannot report completion. Headers sent before body consumption finishes cannot include those later measurements.

`http_dns`, `http_connect`, `http_tls`, `http_conn` and `http_ttfb` describe DNS, dial, TLS, connection acquisition and first-byte timing. `http_conn` counts new and reused connections. `http_wire` counts request writes observed by the transport, including transparent retries. These phases overlap; adding them does not produce request elapsed time.

For a standard HTTP client, set `Transport: middlewares.ServerTiming(baseTransport)`. It requires a non-nil transport. For libraries that own their transport, pass `middlewares.WithClientTimings(ctx)` to their context-aware operation. This records connection phases without replacing their transport or TLS settings. Use the context helper only outside `ServerTiming`, which already installs it. Both helpers live in `github.com/flanksource/commons/http/middlewares` and preserve existing HTTP trace callbacks.

Metric names are fixed and descriptions contain numeric counters. Request URLs, headers, credentials, bodies and connection addresses are not emitted.

## Reuse connections across clients

Create one `TransportPool` for a managed process and close it at shutdown. Call `pool.Client(TransportOptions{Endpoint: endpoint, SecurityNamespace: identity, TLS: tlsOptions})` to obtain a separate client whose raw connection pool can be reused. Configure headers and middleware on that client before concurrent use; keep request HAR and timing collectors outside the shared raw transport.

`TransportPoolOptions.Capacity` defaults to 64; a negative capacity is an error. `Endpoint` must be an absolute HTTP(S) URL without user information. `SecurityNamespace` separates credential/configuration identities; it is hashed with endpoint, TLS and proxy settings and is never logged. `TLS` uses the existing `TLSConfig` fields; `Proxy` optionally supplies an absolute proxy URL, otherwise the default transport's environment proxy behavior applies. Entries use least-recently-used eviction and retain the default idle-connection limits. Eviction and `Close` close idle connections without aborting active exchanges. Requests for a client after pool shutdown fail.

Read response bodies to EOF and close them to enable connection reuse. The Git provider client does this before returning, including when its JSON decoder stops before trailing response data. Credential and TLS changes select different transports; repository and ref data are not cached by this pool.

Run the local TLS and rotation examples with `go test ./http -run TestConnectTimeout -ginkgo.focus 'Managed transport pool'`.
