package logger

type Logger interface {
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Tracef(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	WithValues(keysAndValues ...interface{}) Logger
	IsTraceEnabled() bool
	IsDebugEnabled() bool
	SetLogLevel(level int)
	V(level int) Verbose
}

type Verbose interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Infoln(args ...interface{})
}
