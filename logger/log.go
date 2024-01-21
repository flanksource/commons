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
	IsLevelEnabled(level int) bool
	GetLevel() int
	SetLogLevel(level int)
	SetMinLogLevel(level int)
	V(level int) Verbose
	Named(name string) Logger
	WithoutName() Logger
	WithSkipReportLevel(i int) Logger
}

type Verbose interface {
	Infof(format string, args ...interface{})
	Enabled() bool
}
