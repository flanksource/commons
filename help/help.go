// Package help renders operator-facing documentation for the runtime knobs
// commons owns — log verbosity, log formatting, HTTP wire tracing, HAR capture
// and output formatting — as a clicky Textable, so every CLI built on commons
// can splice the same block into its --help instead of re-documenting the same
// properties (or, more often, not documenting them at all).
//
// The package deliberately sits outside logger/ and http/: clicky imports
// commons/{logger,text,collections,context}, so rendering help from any of
// those packages would create an import cycle.
package help

import (
	"strings"

	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/collections"
)

// Topic is one named section of the help document.
type Topic struct {
	// Name is the stable selector used to filter topics, e.g. "http".
	Name string

	// Title is the section heading.
	Title string

	// Body is the rendered section, excluding its heading.
	Body api.Text
}

// Topics returns every section in display order.
func Topics() []Topic {
	return []Topic{
		{Name: "properties", Title: "Runtime properties", Body: propertiesTopic()},
		{Name: "logging", Title: "Logging verbosity", Body: loggingTopic()},
		{Name: "log-format", Title: "Log formatting", Body: logFormatTopic()},
		{Name: "http", Title: "HTTP wire logging", Body: httpTopic()},
		{Name: "har", Title: "HAR capture", Body: harTopic()},
		{Name: "format", Title: "Output formatting", Body: formatTopic()},
	}
}

// Help composes the selected topics into a single document. With no names every
// topic is included; names are matched with collections.MatchItems, so "http",
// "!har" and "log*" all select as expected. The result is an api.Text, which
// satisfies api.Textable — callers pick String(), ANSI(), Markdown() or HTML().
func Help(names ...string) api.Text {
	doc := api.Text{}
	for _, topic := range Topics() {
		if !collections.MatchItems(topic.Name, names...) {
			continue
		}
		if !doc.IsEmpty() {
			doc = doc.NewLine()
		}
		doc = doc.AddText(topic.Title, styleTitle).NewLine().Add(topic.Body)
	}
	return doc
}

const (
	styleTitle = "font-bold text-blue-400"
	styleCode  = "font-mono text-yellow-600"
	styleMuted = "text-gray-500"

	// knobWidth is the width of the flag/property column, chosen so the widest
	// documented knob still leaves room for its description on an 80 column
	// terminal.
	knobWidth = 32
)

// knob renders an aligned "  <flag or property>  <description>" line.
func knob(name, description string) api.Text {
	return api.Text{}.
		AddText("  "+rpad(name, knobWidth), styleCode).
		AddText(description, styleMuted)
}

// note renders an indented prose line.
func note(text string) api.Text {
	return api.Text{}.AddText("  " + text)
}

// example renders an indented copy-pasteable command fragment.
func example(text string) api.Text {
	return api.Text{}.AddText("    "+text, styleCode)
}

// lines stacks body lines into a single Text, one per line.
func lines(items ...api.Text) api.Text {
	body := api.Text{}
	for _, item := range items {
		body = body.Add(item).NewLine()
	}
	return body
}

func rpad(s string, width int) string {
	if len(s) >= width {
		return s + " "
	}
	return s + strings.Repeat(" ", width-len(s))
}
