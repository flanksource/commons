package logger

import (
	"log/slog"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var log = StandardLogger()
var _ = ginkgo.Describe("LogLevel Parsing", func() {

	ginkgo.It("Logger Names", func() {
		gomega.Expect(camelCaseWords("JohnDoe")).To((gomega.ContainElements("John", "Doe")))
		gomega.Expect(camelCaseWords("johnDoe")).To((gomega.ContainElements("john", "Doe")))
		gomega.Expect(camelCaseWords("john-doe")).To((gomega.ContainElements("john-doe")))

	})
	ginkgo.It("Default log level", func() {
		gomega.Expect(GetLogger().GetLevel()).To(gomega.Equal(Info))
	})

	ginkgo.DescribeTable("parsing log levels from text",
		func(input string, expected LogLevel) {
			level := ParseLevel(log, input)
			gomega.Expect(level).To(gomega.Equal(expected))
		},
		ginkgo.Entry("info level", "info", Info),
		ginkgo.Entry("debug level", "debug", Debug),
		ginkgo.Entry("trace level", "trace", Trace),
		ginkgo.Entry("trace1 level", "trace1", Trace1),
	)

	ginkgo.DescribeTable("parsing log levels from ints",
		func(input int, expected LogLevel) {
			level := ParseLevel(log, input)

			gomega.Expect(level).To(gomega.Equal(expected))
		},

		ginkgo.Entry("error level", -2, Error),
		ginkgo.Entry("warn level", -1, Warn),

		ginkgo.Entry("info level", 0, Info),

		ginkgo.Entry("debug level", 1, Debug),
		ginkgo.Entry("trace level", 2, Trace),
		ginkgo.Entry("trace1 level", 3, Trace1),
		ginkgo.Entry("trace9 level", 9, LogLevel(9)),
	)

	ginkgo.DescribeTable("converting log levels to text",
		func(input LogLevel, expected string) {
			gomega.Expect(input.String()).To(gomega.Equal(expected))
		},
		ginkgo.Entry("info level", Info, "info"),
		ginkgo.Entry("debug level", Trace, "trace"),
		ginkgo.Entry("trace level", Trace1, "trace1"),
	)

	ginkgo.DescribeTable("slog conversion",
		func(input LogLevel, expected slog.Level) {
			gomega.Expect(input.Slog()).To(gomega.Equal(expected))
			gomega.Expect(FromSlogLevel(expected)).To(gomega.Equal(input))
		},
		ginkgo.Entry("info level", Info, slog.LevelInfo),
		ginkgo.Entry("debug level", Debug, slog.LevelDebug),
		ginkgo.Entry("trace level", Trace, SlogTraceLevel),
		ginkgo.Entry("trace1 level", Trace1, SlogTraceLevel-1),
		ginkgo.Entry("trace9 level", LogLevel(9), SlogTraceLevel-7),
	)
})

var _ = ginkgo.Describe("Flag Parsing", func() {
	parse := func(args ...string) flagSet {
		f := flagSet{color: true}
		gomega.Expect(f.parseArgs(args)).To(gomega.Succeed())
		return f
	}

	ginkgo.DescribeTable("verbosity levels",
		func(args []string, expectedLevel string) {
			f := parse(args...)
			gomega.Expect(f.level).To(gomega.Equal(expectedLevel))
		},
		ginkgo.Entry("-v", []string{"-v"}, "1"),
		ginkgo.Entry("-vv", []string{"-vv"}, "2"),
		ginkgo.Entry("-vvv", []string{"-vvv"}, "3"),
		ginkgo.Entry("-vvvvv", []string{"-vvvvv"}, "5"),
		ginkgo.Entry("-v5", []string{"-v5"}, "5"),
		ginkgo.Entry("-v=3", []string{"-v=3"}, "3"),
		ginkgo.Entry("--log-level=debug", []string{"--log-level=debug"}, "debug"),
		ginkgo.Entry("--log-level trace", []string{"--log-level", "trace"}, "trace"),
	)

	ginkgo.DescribeTable("boolean flags",
		func(args []string, json, color, caller bool) {
			f := parse(args...)
			gomega.Expect(f.jsonLogs).To(gomega.Equal(json), "jsonLogs")
			gomega.Expect(f.color).To(gomega.Equal(color), "color")
			gomega.Expect(f.reportCaller).To(gomega.Equal(caller), "reportCaller")
		},
		ginkgo.Entry("--json-logs", []string{"--json-logs"}, true, true, false),
		ginkgo.Entry("--no-color", []string{"--no-color"}, false, false, false),
		ginkgo.Entry("--color=false", []string{"--color=false"}, false, false, false),
		ginkgo.Entry("--report-caller", []string{"--report-caller"}, false, true, true),
		ginkgo.Entry("all flags", []string{"--json-logs", "--no-color", "--report-caller"}, true, false, true),
	)

	ginkgo.DescribeTable("flags mixed with unknown flags",
		func(args []string, json bool, level string) {
			f := parse(args...)
			gomega.Expect(f.jsonLogs).To(gomega.Equal(json), "jsonLogs")
			gomega.Expect(f.level).To(gomega.Equal(level), "level")
		},
		ginkgo.Entry("-Phttp.log=all --json-logs", []string{"-Phttp.log=all", "--json-logs"}, true, ""),
		ginkgo.Entry("--json-logs -Phttp.log=all", []string{"--json-logs", "-Phttp.log=all"}, true, ""),
		ginkgo.Entry("-P http.log=all --json-logs", []string{"-P", "http.log=all", "--json-logs"}, true, ""),
		ginkgo.Entry("-Phttp.log=all -vv", []string{"-Phttp.log=all", "-vv"}, false, "2"),
		ginkgo.Entry("--json-logs -vvv --unknown-flag", []string{"--json-logs", "-vvv", "--unknown-flag"}, true, "3"),
		ginkgo.Entry("run fixture.yaml -Phttp.log=all --json-logs", []string{"run", "fixture.yaml", "-Phttp.log=all", "--json-logs"}, true, ""),
		ginkgo.Entry("-Xfoo=bar --json-logs -vv", []string{"-Xfoo=bar", "--json-logs", "-vv"}, true, "2"),
	)
})

func TestLogger(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Logger Suite")
}
