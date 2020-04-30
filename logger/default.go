package logger

import (
	"fmt"

	"strings"

	"github.com/kr/pretty"
	"github.com/sirupsen/logrus"
)

func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

//Secretf is like Tracef, but attempts to strip any secrets from the text
func Secretf(format string, args ...interface{}) {
	logrus.Tracef(stripSecrets(fmt.Sprintf(format, args...)))
}

//Prettyf is like Tracef, but pretty prints the entire struct
func Prettyf(msg string, obj interface{}) {
	logrus.Tracef(msg, pretty.Sprint(obj))
}

func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

func Tracef(format string, args ...interface{}) {
	logrus.Tracef(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}

func IsTraceEnabled() bool {
	return logrus.IsLevelEnabled(logrus.TraceLevel)
}

func IsDebugEnabled() bool {
	return logrus.IsLevelEnabled(logrus.DebugLevel)
}

func NewLogger(key string, value interface{}) Logger {
	return logrus.New().WithField(key, value)
}

func StandardLogger() Logger {
	return logrus.StandardLogger()
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
