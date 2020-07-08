package logger

type Logger interface {
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Tracef(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	NewLogger(key string, value interface{}) Logger
	NewLoggerWithFields(fields map[string]interface{}) Logger
	IsTraceEnabled() bool
	IsDebugEnabled() bool
	SetLogLevel(level int)
}
