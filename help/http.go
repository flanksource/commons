package help

import (
	"slices"
	"strings"

	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/logger"
)

func httpTopic() api.Text {
	body := lines(
		note("Clients built on commons/http log their traffic on a ladder relative to a base"),
		note("level (debug by default). Credentials in headers, bodies and query strings are"),
		note("redacted before anything is written."),
	)
	body = body.Add(api.NewTableFrom(traceLevelRows())).NewLine()
	return body.Add(lines(
		knob("-Plog.level.http=<level>", "raise the http logger alone, leaving the rest quiet"),
		knob("HTTP_LOG_BASE_LEVEL=<level>", "move the whole ladder; also -Phttp.log.base-level"),
		knob("-Phttp.log.response.body.length", "response prefix bytes logged before a truncation marker (default 4096)"),
		knob("-Phttp.request.maxBufferSize", "bytes of a streamed request body buffered for retry"),
		knob("-Phttp.body.disabled", "never log request or response bodies"),
		knob("-Phttp.headers.disabled", "never log request or response headers"),
		note("CLIs that expose -Phttp.log=<spec> take a spec instead of a level: access,"),
		note("headers, body, request, response, trace or all, plus the additive tokens"),
		note("queryParam, formParams, responseHeaders, tls, timing and auth."),
		note("To keep the exchanges rather than watch them scroll past, see HAR capture."),
	))
}

// traceLevelRow is one rung of the HTTP trace ladder.
type traceLevelRow struct {
	Level    logger.LogLevel
	Flag     string
	Captures string
}

// Columns implements api.TableProvider.
func (traceLevelRow) Columns() []api.ColumnDef {
	return []api.ColumnDef{
		api.Column("level").Label("Level").Style("font-mono").Build(),
		api.Column("flag").Label("Flag").Style("font-mono text-yellow-600").Build(),
		api.Column("captures").Label("Captures").Build(),
	}
}

// Row implements api.TableProvider.
func (r traceLevelRow) Row() map[string]any {
	return map[string]any{
		"level":    r.Level.String(),
		"flag":     r.Flag,
		"captures": r.Captures,
	}
}

// traceLevelRows derives the ladder from http.TraceConfigForLogLevel instead of
// transcribing it, so the table cannot drift from the code and reflects an
// HTTP_LOG_BASE_LEVEL / http.log.base-level override at render time. Levels that
// capture nothing new are folded away, and rungs that only extend the one below
// them are rendered as the delta.
func traceLevelRows() []traceLevelRow {
	var rows []traceLevelRow
	var previous []string
	for level := logger.Warn; level <= logger.Trace4; level++ {
		captures := traceCaptures(http.TraceConfigForLogLevel(level))
		if len(rows) > 0 && slices.Equal(previous, captures) {
			continue
		}
		rows = append(rows, traceLevelRow{
			Level:    level,
			Flag:     levelFlag(level),
			Captures: describeRung(captures, previous),
		})
		previous = captures
	}
	return rows
}

// traceCaptures lists everything a level captures, in ladder order.
func traceCaptures(config http.TraceConfig, enabled bool) []string {
	if !enabled {
		return nil
	}
	if config.AccessLogErrorsOnly {
		return []string{"failed requests only (errors and status >= 400)"}
	}

	captures := []string{"one access line per request"}
	if config.Headers || config.ResponseHeaders {
		captures = append(captures, "headers")
	}
	if config.QueryParam || config.FormParams {
		captures = append(captures, "query and form params")
	}
	if config.Body {
		captures = append(captures, "request bodies")
	}
	if config.TLS {
		captures = append(captures, "TLS summary")
	}
	if config.Response {
		captures = append(captures, "response bodies")
	}
	return captures
}

// describeRung renders captures relative to the rung below it, so each row shows
// what the extra verbosity buys rather than repeating the whole list.
func describeRung(captures, previous []string) string {
	switch {
	case len(captures) == 0:
		return "nothing"
	case len(previous) == 0 || !slices.Equal(previous, captures[:min(len(previous), len(captures))]):
		return strings.Join(captures, ", ")
	default:
		return "+ " + strings.Join(captures[len(previous):], ", ")
	}
}

// levelFlag renders how a level is reached from the command line. logger.Configure
// passes the -v count straight to SetLogLevel, so the count is the level number.
func levelFlag(level logger.LogLevel) string {
	if level <= logger.Info {
		return "--log-level=" + level.String()
	}
	return "-" + strings.Repeat("v", int(level))
}
