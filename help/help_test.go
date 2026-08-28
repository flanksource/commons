package help

import (
	"strings"

	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// documentedKeys are the knobs the help exists to make discoverable. A section
// that silently stops rendering fails here.
var documentedKeys = []string{
	"-P/--properties",
	"--log-level",
	"log.level.<logger>",
	"--json-logs",
	"HTTP_LOG_BASE_LEVEL",
	"http.log.response.body.length",
	"http.har.response.body.length",
	"--format",
}

// renderings covers every output format api.Textable promises, so a section
// that renders in ANSI but collapses to nothing in Markdown is caught.
func renderings(text api.Text) map[string]string {
	return map[string]string{
		"String":   text.String(),
		"ANSI":     text.ANSI(),
		"Markdown": text.Markdown(),
		"HTML":     text.HTML(),
	}
}

var _ = Describe("Topics", func() {
	It("returns the documented sections in display order", func() {
		var names []string
		for _, topic := range Topics() {
			names = append(names, topic.Name)
		}
		Expect(names).To(Equal([]string{"properties", "logging", "log-format", "http", "har", "format"}))
	})

	It("gives every topic a title and a body that renders in every format", func() {
		for _, topic := range Topics() {
			Expect(topic.Title).ToNot(BeEmpty(), topic.Name)
			for format, out := range renderings(topic.Body) {
				Expect(strings.TrimSpace(out)).ToNot(BeEmpty(), "%s topic rendered empty as %s", topic.Name, format)
			}
		}
	})
})

var _ = Describe("Help", func() {
	It("documents every knob in every rendering", func() {
		for format, out := range renderings(Help()) {
			for _, key := range documentedKeys {
				Expect(out).To(ContainSubstring(key), "%s missing from %s output", key, format)
			}
		}
	})

	It("includes each topic's title", func() {
		out := Help().String()
		for _, topic := range Topics() {
			Expect(out).To(ContainSubstring(topic.Title))
		}
	})

	It("selects a single topic by name", func() {
		out := Help("http").String()
		Expect(out).To(ContainSubstring("HTTP_LOG_BASE_LEVEL"))
		Expect(out).ToNot(ContainSubstring("http.har.response.body.length"))
		Expect(out).ToNot(ContainSubstring("--json-logs"))
	})

	It("excludes a topic with a negated name", func() {
		out := Help("!har").String()
		Expect(out).ToNot(ContainSubstring("http.har.response.body.length"))
		Expect(out).To(ContainSubstring("HTTP_LOG_BASE_LEVEL"))
		Expect(out).To(ContainSubstring("--json-logs"))
	})
})

var _ = Describe("the HTTP trace ladder", func() {
	It("describes each level exactly as TraceConfigForLogLevel builds it", func() {
		for level := logger.Warn; level <= logger.Trace4; level++ {
			config, enabled := http.TraceConfigForLogLevel(level)
			captures := strings.Join(traceCaptures(config, enabled), ", ")

			Expect(captures == "").To(Equal(!enabled), "%s claims %q but enabled=%v", level, captures, enabled)
			if !enabled {
				continue
			}
			Expect(strings.Contains(captures, "headers")).To(Equal(config.Headers || config.ResponseHeaders), level.String())
			Expect(strings.Contains(captures, "query and form params")).To(Equal(config.QueryParam || config.FormParams), level.String())
			Expect(strings.Contains(captures, "request bodies")).To(Equal(config.Body), level.String())
			Expect(strings.Contains(captures, "response bodies")).To(Equal(config.Response), level.String())
			Expect(strings.Contains(captures, "TLS summary")).To(Equal(config.TLS), level.String())
		}
	})

	It("folds away levels that capture nothing new", func() {
		var captures []string
		for _, row := range traceLevelRows() {
			captures = append(captures, row.Captures)
		}
		Expect(captures).To(HaveLen(len(dedupeConsecutive(captures))), "consecutive rungs must differ")
	})

	It("renders each extra rung as the delta over the one below it", func() {
		rows := traceLevelRows()
		deltas := map[logger.LogLevel]string{}
		for _, row := range rows {
			if strings.HasPrefix(row.Captures, "+ ") {
				deltas[row.Level] = strings.TrimPrefix(row.Captures, "+ ")
			}
		}
		Expect(deltas).ToNot(BeEmpty(), "the ladder must have at least one incremental rung")

		for level, delta := range deltas {
			below, _ := http.TraceConfigForLogLevel(level - 1)
			config, _ := http.TraceConfigForLogLevel(level)
			added := traceCaptures(config, true)[len(traceCaptures(below, true)):]
			Expect(delta).To(Equal(strings.Join(added, ", ")), level.String())
		}
	})

	It("renders the -v count that reaches each level", func() {
		Expect(levelFlag(logger.Info)).To(Equal("--log-level=info"))
		Expect(levelFlag(logger.Debug)).To(Equal("-v"))
		Expect(levelFlag(logger.Trace)).To(Equal("-vv"))
		Expect(levelFlag(logger.Trace2)).To(Equal("-vvvv"))
	})

	It("shifts with the configured base level", func() {
		bodiesAt := func() logger.LogLevel {
			for _, row := range traceLevelRows() {
				if strings.Contains(row.Captures, "request bodies") {
					return row.Level
				}
			}
			Fail("no rung captures request bodies")
			return logger.Silent
		}

		defaultLevel := bodiesAt()

		properties.Set("http.log.base-level", "trace")
		DeferCleanup(func() { properties.Set("http.log.base-level", "") })

		Expect(bodiesAt()).To(Equal(defaultLevel+1),
			"raising the base level by one must move the whole ladder up by one")
	})
})

// dedupeConsecutive drops runs of equal adjacent entries.
func dedupeConsecutive(values []string) []string {
	var out []string
	for _, v := range values {
		if len(out) > 0 && out[len(out)-1] == v {
			continue
		}
		out = append(out, v)
	}
	return out
}
