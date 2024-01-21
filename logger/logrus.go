package logger

import (
	"fmt"

	"github.com/kr/pretty"
	logrusapi "github.com/sirupsen/logrus"
)

type logrusLogger struct {
	*logrusapi.Entry
	level int
}

type logrusVerbose struct {
	Level logrusapi.Level
	logrusLogger
}

func NewLogrusLogger(existing logrusapi.Ext1FieldLogger, level int) Logger {
	switch v := existing.(type) {
	case *logrusapi.Entry:
		return logrusLogger{Entry: v, level: level}
	case *logrusapi.Logger:
		return logrusLogger{Entry: logrusapi.NewEntry(v), level: level}
	default:
		return logrusLogger{Entry: logrusapi.NewEntry(logrusapi.StandardLogger()), level: level}
	}
}

func (v logrusLogger) WithSkipReportLevel(i int) Logger {
	return v
}

func (v logrusLogger) SetMinLogLevel(level int) {
	if v.GetLevel() >= level {
		return
	}
	v.SetLogLevel(level)
}

func (v logrusLogger) WithoutName() Logger {
	return v
}

func (v logrusLogger) Named(name string) Logger {
	return v.WithValues("name", name)
}

func (v logrusVerbose) Info(args ...interface{}) {
	v.Log(v.Level, args...)
}

func (v logrusVerbose) Infof(format string, args ...interface{}) {
	v.Logf(v.Level, format, args...)
}

func (v logrusVerbose) Infoln(args ...interface{}) {
	v.Logln(v.Level, args...)
}

func (v logrusVerbose) Enabled() bool {
	return v.level >= int(v.Level)
}

func (logrus logrusLogger) V(level int) Verbose {
	var l logrusapi.Level
	switch level {
	case 0:
		l = logrusapi.InfoLevel
	case 1:
		l = logrusapi.DebugLevel
	default:
		l = logrusapi.TraceLevel
	}
	return logrusVerbose{
		logrusLogger: logrus,
		Level:        l,
	}
}

func (logrus logrusLogger) Warnf(format string, args ...interface{}) {
	logrus.Entry.Warnf(format, args...)
}

func (logrus logrusLogger) Infof(format string, args ...interface{}) {
	logrus.Entry.Infof(format, args...)
}

// Secretf is like Tracef, but attempts to strip any secrets from the text
func (logrus logrusLogger) Secretf(format string, args ...interface{}) {
	logrus.Entry.Tracef(stripSecrets(fmt.Sprintf(format, args...)))
}

// Prettyf is like Tracef, but pretty prints the entire struct
func (logrus logrusLogger) Prettyf(msg string, obj interface{}) {
	pretty.Print(obj)
}

func (logrus logrusLogger) Errorf(format string, args ...interface{}) {
	logrus.Entry.Errorf(format, args...)
}

func (logrus logrusLogger) Debugf(format string, args ...interface{}) {
	logrus.Entry.Debugf(format, args...)
}

func (logrus logrusLogger) Tracef(format string, args ...interface{}) {
	logrus.Entry.Tracef(format, args...)
}

func (logrus logrusLogger) Fatalf(format string, args ...interface{}) {
	logrus.Entry.Fatalf(format, args...)
}

func (logrus logrusLogger) WithValues(keysAndValues ...interface{}) Logger {
	fieldMap := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		fieldMap[fmt.Sprintf("%v", keysAndValues[i])] = keysAndValues[i+1]
	}
	return logrusLogger{Entry: logrus.Entry.WithFields(logrusapi.Fields(fieldMap))}
}

func (logrus logrusLogger) GetLevel() int {
	return int(logrus.Entry.Level)
}

func (logrus logrusLogger) IsLevelEnabled(level int) bool {
	return logrus.V(level).Enabled()
}

func (logrus logrusLogger) SetLogLevel(level int) {
	switch {
	case level > 1:
		logrus.Entry.Logger.SetLevel(logrusapi.TraceLevel)
	case level > 0:
		logrus.Entry.Logger.SetLevel(logrusapi.DebugLevel)
	default:
		logrus.Entry.Logger.SetLevel(logrusapi.InfoLevel)
	}
}

func (logrus logrusLogger) IsTraceEnabled() bool {
	return logrus.Entry.Logger.IsLevelEnabled(logrusapi.TraceLevel)
}

func (logrus logrusLogger) IsDebugEnabled() bool {
	return logrus.Entry.Logger.IsLevelEnabled(logrusapi.DebugLevel)
}
