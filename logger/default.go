package logger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/pflag"

	"github.com/flanksource/commons/properties"
)

var currentLogger Logger
var flags = &flagSet{}

type flagSet struct {
	color, reportCaller, jsonLogs, logToStderr bool
	level                                      string
}

func (f flagSet) String() string {
	return fmt.Sprintf("level=%v json=%v color=%v caller=%v stderr=%v", f.level, f.jsonLogs, f.color, f.reportCaller, f.logToStderr)
}

func (f *flagSet) bindFlags(flags *pflag.FlagSet) {
	_ = flags.CountP("log-level", "v", "Increase logging level")
	flags.BoolVar(&f.jsonLogs, "json-logs", false, "Print logs in json format to stderr")
	flags.BoolVar(&f.color, "color", true, "Print logs using color")
	flags.BoolVar(&f.reportCaller, "report-caller", false, "Report log caller info")
	flags.BoolVar(&f.logToStderr, "log-to-stderr", false, "Log to stderr instead of stdout")
}

func (f *flagSet) Parse() error {
	logFlagset := pflag.NewFlagSet("logger", pflag.ContinueOnError)
	logFlagset.ParseErrorsAllowlist = pflag.ParseErrorsAllowlist{UnknownFlags: true}

	// standalone parsing of flags to ensure we always have the correct values
	f.bindFlags(logFlagset)
	if err := logFlagset.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return nil
		}
		return err
	}

	re, _ := regexp.Compile("-v{1,}")
	for _, arg := range os.Args[1:] {
		// FIXME there seems to be a race condition where pflag

		// will return a count that does not match the actual number of -v flags
		if strings.HasPrefix(arg, "-v") {
			if strings.Contains(arg, "=") {
				f.level = arg[3:]
			} else if ok := re.MatchString(arg); ok {
				f.level = fmt.Sprintf("%d", len(arg)-1)
			} else {
				f.level = arg[2:]
			}
		}
	}
	return nil
}

func init() {
	UseSlog()
}

func IsJsonLogs() bool {
	return flags.jsonLogs
}

type Flags struct {
	Color, ReportCaller, JsonLogs, LogToStderr bool
	Level                                      string
	LevelCount                                 int
}

func Configure(flags Flags) {
	// Get the verbosity count from the parsed flags
	if flags.LevelCount > 0 {
		currentLogger.SetLogLevel(flags.LevelCount)
	} else if level := flags.Level; level != "" && level != "info" {
		currentLogger.SetLogLevel(level)
	}

	// Apply other flags
	if flags.JsonLogs {
		properties.Set("log.json", "true")
	}

	if flags.ReportCaller {
		properties.Set("log.report.caller", "true")
	}

	if flags.LogToStderr {
		properties.Set("log.stderr", "true")
	}
	if !flags.Color {
		properties.Set("log.color", "false")
	}

	currentLogger = *New("")

}

// BindFlags add flags to an existing flag set,
// note that this is not an actual binding which occurs later during initialization
func BindFlags(flags *pflag.FlagSet) {
	flags.CountP("loglevel", "v", "Increase logging level")
	flags.String("log-level", "info", "Set the default log level")
	flags.Bool("json-logs", false, "Print logs in json format to stderr")
	flags.Bool("color", true, "Print logs using color")
	flags.Bool("report-caller", false, "Report log caller info")
	flags.Bool("log-to-stderr", false, "Log to stderr instead of stdout")
}

// UseCobraFlags initializes the logger using values from parsed cobra flags.
// This should be called after cobra has parsed the command line arguments.
func UseCobraFlags(flags *pflag.FlagSet) {
	// Get the verbosity count from the parsed flags
	if v, err := flags.GetCount("loglevel"); err == nil && v > 0 {
		currentLogger.SetLogLevel(v)
	} else if level, err := flags.GetString("log-level"); err == nil && level != "" && level != "info" {
		currentLogger.SetLogLevel(level)
	}

	// Apply other flags
	if jsonLogs, err := flags.GetBool("json-logs"); err == nil && jsonLogs {
		currentLogger = New("")
	}
}

// Warnf logs a warning message with formatting support.
// These are messages about potentially harmful situations.
func Warnf(format string, args ...interface{}) {
	currentLogger.Warnf(format, args...)
}

// Infof logs an informational message with formatting support.
// These are general informational messages about normal operations.
func Infof(format string, args ...interface{}) {
	currentLogger.Infof(format, args...)
}

// Secretf logs a trace message after attempting to strip any secrets from the text.
// It automatically redacts common secret patterns like passwords, tokens, and API keys.
//
// Example:
//
//	logger.Secretf("Connecting with password=%s", password) // password will be redacted
func Secretf(format string, args ...interface{}) {
	currentLogger.Tracef(StripSecrets(fmt.Sprintf(format, args...)))
}

// Prettyf logs a trace message with a pretty-printed representation of the given object.
// Useful for debugging complex data structures.
//
// Example:
//
//	logger.Prettyf("User data:", userStruct) // Logs formatted struct
func Prettyf(msg string, obj interface{}) {
	currentLogger.Tracef(msg, Pretty(obj))
}

// Errorf logs an error message with formatting support.
// Use this for errors that need attention but don't terminate the program.
func Errorf(format string, args ...interface{}) {
	currentLogger.Errorf(format, args...)
}

// Debugf logs a debug message with formatting support.
// These messages are only shown when debug logging is enabled.
func Debugf(format string, args ...interface{}) {
	currentLogger.Debugf(format, args...)
}

// Tracef logs a trace message with formatting support.
// These are very detailed messages for troubleshooting, only shown at trace level.
func Tracef(format string, args ...interface{}) {
	currentLogger.Tracef(format, args...)
}

// Fatalf logs a fatal error message and terminates the program.
// Use this for unrecoverable errors.
func Fatalf(format string, args ...interface{}) {
	currentLogger.Fatalf(format, args...)
}

// V returns a verbose logger for conditional logging at the specified level.
// The level can be an integer or a LogLevel constant.
//
// Example:
//
//	logger.V(2).Infof("Detailed info") // Only logs at verbosity 2+
func V(level any) Verbose {
	return currentLogger.V(level)
}

// IsTraceEnabled returns true if trace level logging is enabled.
func IsTraceEnabled() bool {
	return currentLogger.IsTraceEnabled()
}

// IsLevelEnabled returns true if the specified verbosity level is enabled.
//
// Example:
//
//	if logger.IsLevelEnabled(3) {
//	    // Perform expensive operation only if logging at level 3
//	}
func IsLevelEnabled(level int) bool {
	return currentLogger.V(level).Enabled()
}

// IsDebugEnabled returns true if debug level logging is enabled.
func IsDebugEnabled() bool {
	return currentLogger.IsDebugEnabled()
}

// WithValues returns a new logger with additional key-value pairs.
// These values will be included in all log messages from the returned logger.
//
// Example:
//
//	userLogger := logger.WithValues("user_id", 123, "session", "abc")
//	userLogger.Infof("User action") // Logs with user_id=123 session=abc
func WithValues(keysAndValues ...interface{}) Logger {
	return currentLogger.WithValues(keysAndValues...)
}

// SetLogger replaces the global logger instance.
// Use this to configure a custom logger implementation.
func SetLogger(logger Logger) {
	currentLogger = logger
}

// Use configures the logger to write to the specified writer.
// This replaces the current logger with one that outputs to the given writer.
// Useful for integrating with test frameworks like Ginkgo.
//
// Example:
//
//	logger.Use(GinkgoWriter) // Route logger output to Ginkgo's test writer
func Use(writer io.Writer) {
	currentLogger = NewWithWriter(writer)
}

// StandardLogger returns the current global logger instance.
// This is equivalent to GetLogger().
func StandardLogger() Logger {
	return currentLogger
}
