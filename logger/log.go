package logger

import "log/slog"

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
	IsLevelEnabled(level LogLevel) bool
	GetLevel() LogLevel
	SetLogLevel(level any)
	SetMinLogLevel(level any)
	V(level any) Verbose
	WithV(level any) Logger
	Named(name string) Logger
	WithoutName() Logger
	WithSkipReportLevel(i int) Logger
	GetSlogLogger() *slog.Logger
}

type Verbose interface {
	Infof(format string, args ...interface{})
	Enabled() bool
}

type LogLevel int

const (
	Debug  LogLevel = 1
	Trace  LogLevel = 2
	Trace1 LogLevel = 3
	Trace2 LogLevel = 4
	Trace3 LogLevel = 5
	Trace4 LogLevel = 6
	Info   LogLevel = 0
	Warn   LogLevel = -1
	Error  LogLevel = -2
	Fatal  LogLevel = -3
	Silent LogLevel = 10
)

const (
	cyan      = "\x1b[36"
	Cyan      = cyan + Normal
	magenta   = "\x1b[35"
	Magenta   = magenta + Normal
	DarkWhite = "\x1b[38;5;244m"
	Normal    = "m"
	Reset     = "\x1b[0m"
)
