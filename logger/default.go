package logger

import "github.com/sirupsen/logrus"

func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
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
