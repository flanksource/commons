package help

import "github.com/flanksource/clicky/api"

func harTopic() api.Text {
	return lines(
		note("A HAR file records whole request/response pairs — including OAuth token fetches,"),
		note("redirect hops and retries — for import into browser DevTools or any HAR viewer."),
		note("Naming an output path is what turns capture on:"),
		example("-Phttp.har=trace.har"),
		knob("-Phttp.har=<path>", "capture to <path>; unset means no capture"),
		knob("-Phttp.<feature>.har=<path>", "capture one subsystem to its own file"),
		knob("-Phttp.har.level=metadata", "headers, query and timings only; default is full bodies"),
		knob("-Phttp.har.sensitive=true", "keep credentials verbatim so the archive replays"),
		knob("-Phttp.har.response.body.length", "bytes captured per request/response body (default 4194304, 0 = uncapped)"),
		note("Bodies larger than the cap retain their prefix and are marked as truncated."),
		note("Bodies are captured for application/json and application/x-www-form-urlencoded"),
		note("only. By default headers, body keys and query strings are redacted with the same"),
		note("rules as the wire log; http.har.sensitive turns that off and writes the file 0600."),
	)
}
