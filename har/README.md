# HAR capture and optional integrations

`github.com/flanksource/commons/har` and `github.com/flanksource/commons/http/middlewares` use only the Go standard library and other standard-library-only commons packages. Embeddable clients can capture traffic without importing AWS, Clicky, OAuth, Prometheus, or OpenTelemetry.

```go
collector := har.NewCollector(har.DefaultConfig())
client := &http.Client{Transport: collector.Middleware()(http.DefaultTransport)}
```

`HARConfig` controls the retained entry count, body size, content types, and redaction. `har.DefaultConfig()` reads the existing `http.har.response.body.length` and `http.har.sensitive` properties. `har.Entry`, `har.Request`, `har.Response`, `har.Collector`, and their JSON representations retain their existing fields and capture behavior.

Optional features live in separate packages:

| Feature | Import under `github.com/flanksource/commons/` | Usage |
|---|---|---|
| AWS SigV4 signing | `http/middlewares/awssigv4` | `awssigv4.NewAWSSigv4Transport(config, transport)` |
| OAuth client credentials | `http/middlewares/oauth` | `oauth.NewOauthTransport(config).RoundTripper` |
| HTTP trace and access logs | `http/middlewares/tracing` | `tracing.NewTracedTransport(config)` or `tracing.NewLogger(config)` |
| Clicky server timing | `http/middlewares/timing` | `timing.ServerTiming(transport)` or `timing.WithClientTimings(ctx)` |
| Clicky HAR rendering | `har/harpretty` | `harpretty.Entry(entry).Pretty()` |
| Prometheus queue metrics | `collections/prommetrics` | `prommetrics.New(prommetrics.Options[T]{Name: "work"})` |
| YAML formatting and highlighting | `logger/yamlcolor` | Blank import to register the formatter |

Core HAR types no longer implement Clicky interfaces. Convert entries at the rendering boundary; use the collector adapter for summaries and tables:

```go
view := &harpretty.Collector{Collector: collector}
summary := view.Pretty()
table := view.Table()
detail := harpretty.Entry(entry).Detail(harpretty.DetailOptions{
    RequestHeaders: true,
    ResponseBody:   true,
})
```

The adapters implement the existing `Pretty`, `Columns`, `Row`, `RowDetail`, `Detail`, and collector `Table` methods. `DetailOptions` also belongs to `harpretty`. Importing an optional package pulls in that feature's dependencies; importing core HAR capture does not.

`har.NewRegistry`, `logger.SanitizeHeaders`, `properties.BindFlags`, `properties.LoadFile`, `Properties.Watch`, and `collections.NewQueue` keep their original package paths. Property overrides remain process-wide, and the most recently parsed flag set retains precedence. File watching polls mtime, size, and file identity every `properties.WatchInterval` (100 ms); the close function returned by `Watch()` is idempotent.

Queue metrics are disabled when `QueueOpts.Metrics` is nil. To opt in, pass `Metrics: prommetrics.New(prommetrics.Options[T]{Name: "work"})`. The metrics interface receives enqueue, dequeue, deduplication, and size updates; the Prometheus adapter retains the existing metric names and labels. Importing `logger/yamlcolor` registers YAML serialization and highlighting. Otherwise `PrettyYAML` leaves YAML text uncoloured, and `Pretty` formats structured values as JSON.

Existing consumers must move the four optional middleware imports, wrap HAR values before rendering, and use commons-owned SigV4 credentials. NTLM authentication has been removed. See the [SigV4 package guide](../http/middlewares/awssigv4/README.md) for the AWS-provider adapter, signing options, and SDK comparison commands. The higher-level `commons/http` client continues to install its supported auth, tracing, logging, HAR, and timing middleware.

Run the local capture and rendering example:

```sh
go run ./examples/har
```

The example starts a local HTTP test server and captures one request. Run the dependency regression test with:

```sh
GOWORK=off go test ./test/dependencies -count=1 -v
```

The guard runs `go list -deps` for both core packages and rejects every module except `github.com/flanksource/commons`; standard-library packages are allowed.
