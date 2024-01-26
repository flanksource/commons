package logger

import (
	"flag"
	"fmt"
	"strings"

	"github.com/kr/pretty"
	logsrusapi "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var currentLogger Logger
var color, reportCaller, jsonLogs bool
var level int

func init() {
	currentLogger = newZap(1)
}

func IsJsonLogs() bool {
	return jsonLogs
}

func BindFlags(flags *pflag.FlagSet) {
	flags.CountVarP(&level, "loglevel", "v", "Increase logging level")
	flags.BoolVar(&jsonLogs, "json-logs", false, "Print logs in json format to stderr")
	flags.BoolVar(&color, "color", true, "Print logs using color")
	flags.BoolVar(&reportCaller, "report-caller", false, "Report log caller info")
}

func BindGoFlags() {
	flag.IntVar(&level, "v", 0, "Increase logging level")
	flag.BoolVar(&jsonLogs, "json-logs", false, "Print logs in json format to stderr")
	flag.BoolVar(&color, "color", true, "Print logs using color")
	flag.BoolVar(&reportCaller, "report-caller", false, "Report log caller info")
}

func UseLogsrus() {
	logger := logsrusapi.StandardLogger()
	if jsonLogs {
		logger.SetFormatter(&logsrusapi.JSONFormatter{})
	} else {
		logger.SetFormatter(&logsrusapi.TextFormatter{
			DisableColors: !color,
			ForceColors:   color,
			FullTimestamp: true,
			DisableQuote:  true,
		})
	}
	currentLogger = NewLogrusLogger(logger, level)
	currentLogger.SetLogLevel(level)
}

func Warnf(format string, args ...interface{}) {
	currentLogger.Warnf(format, args...)
}

func Infof(format string, args ...interface{}) {
	currentLogger.Infof(format, args...)
}

// Secretf is like Tracef, but attempts to strip any secrets from the text
func Secretf(format string, args ...interface{}) {
	currentLogger.Tracef(stripSecrets(fmt.Sprintf(format, args...)))
}

// Prettyf is like Tracef, but pretty prints the entire struct
func Prettyf(msg string, obj interface{}) {
	pretty.Print(obj)
	// currentLogger.Tracef(msg, pretty.Sprint(obj))
}

func Errorf(format string, args ...interface{}) {
	currentLogger.Errorf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	currentLogger.Debugf(format, args...)
}

func Tracef(format string, args ...interface{}) {
	currentLogger.Tracef(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	currentLogger.Fatalf(format, args...)
}
func V(level int) Verbose {
	return currentLogger.V(level)
}

func IsTraceEnabled() bool {
	return currentLogger.IsTraceEnabled()
}

func IsLevelEnabled(level int) bool {
	return currentLogger.V(level).Enabled()
}

func IsDebugEnabled() bool {
	return currentLogger.IsDebugEnabled()
}

func WithValues(keysAndValues ...interface{}) Logger {
	return currentLogger.WithValues(keysAndValues...)
}

func SetLogger(logger Logger) {
	currentLogger = logger
}

func StandardLogger() Logger {
	return currentLogger
}

// stripSecrets takes a YAML or INI formatted text and removes any potentially secret data
// as denoted by keys containing "pass" or "secret" or exact matches for "key"
// the last character of the secret is kept to aid in troubleshooting
func stripSecrets(text string) string {
	out := ""
	for _, line := range strings.Split(text, "\n") {

		var k, v, sep string
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			k = parts[0]
			if len(parts) > 1 {
				v = parts[1]
			}
			sep = ":"
		} else if strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			k = parts[0]
			if len(parts) > 1 {
				v = parts[1]
			}
			sep = "="
		} else {
			v = line
		}

		if strings.Contains(k, "pass") || strings.Contains(k, "secret") || strings.Contains(k, "_key") || strings.TrimSpace(k) == "key" || strings.TrimSpace(k) == "token" {
			if len(v) == 0 {
				out += k + sep + "\n"
			} else {
				out += k + sep + "****" + v[len(v)-1:] + "\n"
			}
		} else {
			out += k + sep + v + "\n"
		}
	}
	return out

}
