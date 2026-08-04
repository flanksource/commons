package help

import "github.com/flanksource/clicky/api"

func formatTopic() api.Text {
	return lines(
		knob("--format=<format>", "pretty (default), json, yaml, csv, html, markdown, pdf, slack"),
		knob("--json, --yaml, --csv, --pdf", "shorthands for the matching --format value"),
		knob("--markdown, --html, --pretty", "the remaining --format shorthands"),
		knob("--tree, --table", "display structure, additive with the chosen format"),
		knob("--filter=<cel>", "CEL expression filtering the data before rendering"),
		knob("--no-color", "disable ANSI colour in rendered output"),
		note("--format also takes a comma separated list of format=file sinks, which writes"),
		note("each rendering to its own file instead of stdout:"),
		example("--format=json=report.json,markdown=summary.md"),
		note("These flags come from clicky; a CLI accepts them only if it binds that group."),
	)
}
