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

func TestLogger(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Logger Suite")
}
