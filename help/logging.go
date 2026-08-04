package help

import "github.com/flanksource/clicky/api"

func propertiesTopic() api.Text {
	return lines(
		note("Every log.* and http.* setting below is a property. Properties are set with"),
		note("-P/--properties, which is repeatable and also accepts comma separated pairs:"),
		example("-Plog.level.http=trace,http.body.disabled=true"),
	)
}

func loggingTopic() api.Text {
	return lines(
		knob("-v, -vv, -vvv, -vvvv", "raise the global level to debug, trace, trace1, trace2"),
		knob("--log-level=<level>", "error, warn, info (default), debug, trace, trace1..trace4"),
		knob("LOG_LEVEL=<level>", "the same levels, from the environment"),
		knob("-Plog.level.<logger>=<level>", "raise a single subsystem, e.g. log.level.http=trace"),
		note("A -v count wins over --log-level. Named loggers inherit the global level unless"),
		note("log.level.<logger> overrides them, so one subsystem can be traced without the"),
		note("rest of the process becoming verbose."),
	)
}

func logFormatTopic() api.Text {
	return lines(
		knob("--json-logs", "structured JSON on stderr instead of console text"),
		knob("--report-caller", "prefix each line with its source file and line"),
		knob("--color=false / --no-color", "disable ANSI colour (whichever flag the CLI binds)"),
		knob("NO_COLOR, COLOR=no, TERM=dumb", "disable ANSI colour from the environment"),
		note("Logs always go to stderr; --log-to-stderr is accepted but ignored."),
	)
}
