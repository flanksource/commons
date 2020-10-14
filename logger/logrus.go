package logger

import (
	"fmt"

	"github.com/kr/pretty"
	logrusapi "github.com/sirupsen/logrus"
)

type logrusLogger struct {
	*logrusapi.Logger
}

func (logrus logrusLogger) Warnf(format string, args ...interface{}) {
	logrus.Logger.Warnf(format, args...)
}

func (logrus logrusLogger) Infof(format string, args ...interface{}) {
	logrus.Logger.Infof(format, args...)
}

//Secretf is like Tracef, but attempts to strip any secrets from the text
func (logrus logrusLogger) Secretf(format string, args ...interface{}) {
	logrus.Logger.Tracef(stripSecrets(fmt.Sprintf(format, args...)))
}

//Prettyf is like Tracef, but pretty prints the entire struct
func (logrus logrusLogger) Prettyf(msg string, obj interface{}) {
	pretty.Print(obj)
}

func (logrus logrusLogger) Errorf(format string, args ...interface{}) {
	logrus.Logger.Errorf(format, args...)
}

func (logrus logrusLogger) Debugf(format string, args ...interface{}) {
	logrus.Logger.Debugf(format, args...)
}

func (logrus logrusLogger) Tracef(format string, args ...interface{}) {
	logrus.Logger.Tracef(format, args...)
}

func (logrus logrusLogger) Fatalf(format string, args ...interface{}) {
	logrus.Logger.Fatalf(format, args...)
}

func (logrus logrusLogger) NewLogger(key string, value interface{}) Logger {
	return logrusLogger{Logger: logrusapi.New().WithField(key, value).Logger}
}

func (logrus logrusLogger) NewLoggerWithFields(fields map[string]interface{}) Logger {
	return logrusLogger{Logger: logrusapi.New().WithFields(logrusapi.Fields(fields)).Logger}
}

func NewLogrusLogger(existing logrusapi.Logger) Logger {
	return logrusLogger{Logger: &existing}
}

func (logrus logrusLogger) SetLogLevel(level int) {
	switch {
	case level > 1:
		logrus.Logger.SetLevel(logrusapi.TraceLevel)
	case level > 0:
		logrus.Logger.SetLevel(logrusapi.DebugLevel)
	default:
		logrus.Logger.SetLevel(logrusapi.InfoLevel)
	}
}

func (logrus logrusLogger) IsTraceEnabled() bool {
	return logrus.Logger.IsLevelEnabled(logrusapi.TraceLevel)
}

func (logrus logrusLogger) IsDebugEnabled() bool {
	return logrus.Logger.IsLevelEnabled(logrusapi.DebugLevel)
}
