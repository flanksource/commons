package logger

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

var currentLogger Logger
var color, reportCaller, jsonLogs bool
var level int

func init() {
	UseSlog()
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

func Warnf(format string, args ...interface{}) {
	currentLogger.Warnf(format, args...)
}

func Infof(format string, args ...interface{}) {
	currentLogger.Infof(format, args...)
}

// Secretf is like Tracef, but attempts to strip any secrets from the text
func Secretf(format string, args ...interface{}) {
	currentLogger.Tracef(StripSecrets(fmt.Sprintf(format, args...)))
}

// Prettyf is like Tracef, but pretty prints the entire struct
func Prettyf(msg string, obj interface{}) {
	currentLogger.Tracef(msg, Pretty(obj))
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
func V(level any) Verbose {
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

func PrintableSecret(secret string) string {
	if len(secret) == 0 {
		return "<nil>"
	} else if len(secret) > 30 {
		sum := md5.Sum([]byte(secret))
		hash := hex.EncodeToString(sum[:])
		return fmt.Sprintf("md5(%s),length=%d", hash[0:8], len(secret))
	} else if len(secret) > 16 {
		return fmt.Sprintf("%s****%s", secret[0:1], secret[len(secret)-2:])
	} else if len(secret) > 10 {
		return fmt.Sprintf("****%s", secret[len(secret)-1:])
	}
	return "****"
}

// StripSecrets takes a YAML or INI formatted text and removes any potentially secret data
// as denoted by keys containing "pass" or "secret" or exact matches for "key"
// the last character of the secret is kept to aid in troubleshooting
func StripSecrets(text string) string {
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
