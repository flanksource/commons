package logger

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
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
	// standalone parsing of flags to ensure we always have the correct values
	f.bindFlags(logFlagset)
	logFlagset.ParseErrorsWhitelist.UnknownFlags = true
	if err := logFlagset.Parse(os.Args[1:]); err != nil {
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

func Warnf(format string, args ...interface{}) {
	currentLogger.Warnf(format, args...)
}

func Infof(format string, args ...interface{}) {
	currentLogger.Infof(format, args...)
}

// Secretf is like Tracef, but attempts to strip any secrets from the text
func Secretf(format string, args ...interface{}) {
	currentLogger.Tracef(StripSecrets(fmt.Sprintf(format, args...)))
}

// Prettyf is like Tracef, but pretty prints the entire struct
func Prettyf(msg string, obj interface{}) {
	currentLogger.Tracef(msg, Pretty(obj))
}

func Errorf(format string, args ...interface{}) {
	currentLogger.Errorf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	currentLogger.Debugf(format, args...)
}

func Tracef(format string, args ...interface{}) {
	currentLogger.Tracef(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	currentLogger.Fatalf(format, args...)
}
func V(level any) Verbose {
	return currentLogger.V(level)
}

func IsTraceEnabled() bool {
	return currentLogger.IsTraceEnabled()
}

func IsLevelEnabled(level int) bool {
	return currentLogger.V(level).Enabled()
}

func IsDebugEnabled() bool {
	return currentLogger.IsDebugEnabled()
}

func WithValues(keysAndValues ...interface{}) Logger {
	return currentLogger.WithValues(keysAndValues...)
}

func SetLogger(logger Logger) {
	currentLogger = logger
}

func StandardLogger() Logger {
	return currentLogger
}
