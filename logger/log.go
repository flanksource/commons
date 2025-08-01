// Package logger provides a flexible logging interface with support for
// multiple backends (logrus, slog) and various output formats.
//
// The package offers a global logger instance that can be configured with
// different log levels, output formats, and additional context values.
//
// Basic Usage:
//
//	logger.Infof("Server started on port %d", 8080)
//	logger.Debugf("Processing request: %s", requestID)
//	logger.Errorf("Failed to connect to database: %v", err)
//
// With Context Values:
//
//	log := logger.GetLogger().WithValues("user", userID, "request", requestID)
//	log.Infof("Processing user request")
//
// Named Loggers:
//
//	dbLogger := logger.GetLogger().Named("database")
//	apiLogger := logger.GetLogger().Named("api")
//
// Log Levels:
//
//	logger.SetLogLevel(logger.Debug)  // Enable debug logging
//	logger.SetLogLevel(logger.Trace)  // Enable trace logging
//
// The package supports standard log levels (Info, Debug, Error, etc.) plus
// extended trace levels (Trace, Trace1-4) for fine-grained debugging.
package logger

import (
	"fmt"
	"io"
	"log/slog"
)

// Logger is the main interface for logging operations.
// It provides methods for different log levels and supports
// structured logging with key-value pairs.
type Logger interface {
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Tracef(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	// WithValues returns a new Logger with additional key-value pairs
	// that will be included in all subsequent log messages.
	//
	// Example:
	//   log := logger.WithValues("component", "auth", "version", "1.0")
	//   log.Infof("User logged in") // Will include component=auth version=1.0
	WithValues(keysAndValues ...interface{}) Logger
	IsTraceEnabled() bool
	IsDebugEnabled() bool
	IsLevelEnabled(level LogLevel) bool
	GetLevel() LogLevel
	SetLogLevel(level any)
	SetMinLogLevel(level any)
	// V returns a Verbose logger that only logs if the specified level is enabled.
	// Level can be an integer or a named level (Debug, Trace, etc.).
	//
	// Example:
	//   logger.V(2).Infof("This only logs at verbosity 2+")
	//   logger.V(logger.Trace).Infof("This only logs at trace level")
	V(level any) Verbose
	WithV(level any) Logger
	// Named returns a new Logger with the specified name added to the logging context.
	// This helps identify which component or subsystem generated the log.
	//
	// Example:
	//   dbLogger := logger.Named("database")
	//   dbLogger.Infof("Connected to database") // Logs with name="database"
	Named(name string) Logger
	WithoutName() Logger
	WithSkipReportLevel(i int) Logger
	GetSlogLogger() *slog.Logger
}

// Verbose provides conditional logging based on verbosity levels.
// It's returned by Logger.V() and only logs if the specified level is enabled.
//
// Example:
//
//	// Only logs if verbosity is 2 or higher
//	logger.V(2).Infof("Detailed debug information: %v", data)
type Verbose interface {
	io.Writer
	Infof(format string, args ...interface{})
	WithFilter(filters ...string) Verbose
	Enabled() bool
}

// LogLevel represents the severity of a log message.
// Higher values indicate more verbose logging.
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

func (l LogLevel) String() string {
	switch l {
	case Debug:
		return "debug"
	case Trace:
		return "trace"
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Error:
		return "error"
	case Fatal:
		return "fatal"
	case Silent:
		return "silent"
	}
	return fmt.Sprintf("trace%d", l-Trace)
}

const (
	cyan      = "\x1b[36"
	Cyan      = cyan + Normal
	magenta   = "\x1b[35"
	Magenta   = magenta + Normal
	DarkWhite = "\x1b[38;5;244m"
	Normal    = "m"
	Reset     = "\x1b[0m"
)
