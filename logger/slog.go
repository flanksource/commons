package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/flanksource/commons/is"
	"github.com/flanksource/commons/properties"
	"github.com/kr/pretty"
	"github.com/lmittmann/tint"
	"github.com/lrita/cmap"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
)

var (
	isTTY = is.TTY()
)

const rootName = "root"

var logFlagset = pflag.NewFlagSet("logger", pflag.ContinueOnError)

var namedLoggers cmap.Map[string, *SlogLogger]
var todo = context.TODO()

func GetNamedLoggingLevels() (levels map[string]string) {
	levels = make(map[string]string)
	namedLoggers.Range(func(key string, value *SlogLogger) bool {
		levels[key] = FromSlogLevel(value.Level.Level()).String()
		return true
	})
	return levels
}

func BrightF(msg string, args ...interface{}) string {
	if isTTY && color && !jsonLogs {
		return DarkWhite + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

var SlogTraceLevel slog.Level = slog.LevelDebug - 1
var SlogFatal = slog.LevelError + 1

func GetSlogLogger() SlogLogger {
	return currentLogger.(SlogLogger)
}

func onPropertyUpdate(props *properties.Properties) {
	for k, v := range props.GetAll() {
		if k == "log.level" || k == "log.json" || k == "log.caller" || k == "log.color" {
			root := New(rootName)
			existing := GetLogger()
			(*existing).Logger = root.Logger
		} else if k == "db.log.level" {
			GetLogger("db").SetLogLevel(v)
		} else if strings.HasPrefix(k, "log.level") {
			name := strings.TrimPrefix(k, "log.level.")
			named := GetLogger(strings.Split(name, ".")...)
			named.SetLogLevel(v)
		} else if k == "log.report.caller" {
			reportCaller, _ = strconv.ParseBool(v)
		}
	}

	if props.On(false, "log.json") && props.On(false, "log.color") {
		// disable color logs when json logs are enabled
		properties.Set("log.color", "false")
	}
}

func New(prefix string) *SlogLogger {
	// create a new slogger
	var logger *SlogLogger
	var lvl = &slog.LevelVar{}

	reportCaller := properties.On(reportCaller, fmt.Sprintf("log.caller.%s", prefix), "log.caller")
	logJson := properties.On(jsonLogs, "log.json")
	logColor := properties.On(color, fmt.Sprintf("log.color.%s", prefix), "log.color")

	var rootLevel string
	cmdLineLevel := logFlagset.Lookup("loglevel")
	if cmdLineLevel.Changed {
		rootLevel = LogLevel(level).String()
	} else {
		rootLevel = properties.String("info", "log.level")
	}
	namedLevel := properties.String(rootLevel, "log.level."+prefix)

	logStderr := properties.On(logToStderr, "log.stderr")
	destination := os.Stdout
	if logStderr {
		destination = os.Stderr
	}

	if logJson {
		color = false
		jsonLogs = true
		logger = &SlogLogger{
			Level: lvl,
			Logger: slog.New(slog.NewJSONHandler(destination, &slog.HandlerOptions{
				AddSource: reportCaller,
				Level:     lvl,
			})),
		}

	} else {
		logger = &SlogLogger{
			Logger: slog.New(tint.NewHandler(destination, &tint.Options{
				Level:      lvl,
				NoColor:    !logColor,
				AddSource:  reportCaller,
				TimeFormat: properties.String("15:04:05.999", fmt.Sprintf("log.time.format.%s", prefix), "log.time.format"),
			})),
			Level: lvl,
		}
	}

	if prefix != "" && prefix != rootName {
		logger.Prefix = prefix
	}

	logger.SetLogLevel(namedLevel)
	return logger
}
func UseSlog() {
	if currentLogger != nil {
		return
	}

	logFlagset = pflag.NewFlagSet("logger", pflag.ContinueOnError)
	// standalone parsing of flags to ensure we always have the correct values
	bindFlags(logFlagset)
	logFlagset.ParseErrorsWhitelist.UnknownFlags = true
	if err := logFlagset.Parse(os.Args[1:]); err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("level=%d json=%v color=%v caller=%v args=%s ", level, jsonLogs, color, reportCaller, strings.Join(os.Args[1:], " "))

	root := New(rootName)

	slog.SetDefault(root.Logger)
	namedLoggers.Store(rootName, root)
	currentLogger = root

	properties.RegisterListener(onPropertyUpdate)
}

func camelCaseWords(s string) []string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsUpper(r) {
			result.WriteRune(' ')
			result.WriteRune(r)

		} else {
			result.WriteRune(r)
		}
	}
	return strings.Fields(result.String())
}

func GetLogger(names ...string) *SlogLogger {
	parent, _ := namedLoggers.Load(rootName)
	if len(names) == 0 {
		return parent
	}

	path := ""
	for i, name := range names {
		name = strings.ToLower(strings.Join(camelCaseWords(name), " "))
		if path != "" {
			path += "."
		}
		path = path + strings.TrimSpace(name)
		if v, ok := namedLoggers.Load(path); ok {
			return v
		}
		if i == 0 {
			break
		}
	}
	child, _ := namedLoggers.LoadOrStore(path, New(path))
	return child
}

type SlogLogger struct {
	*slog.Logger
	Prefix    string
	Level     *slog.LevelVar
	Parent    *SlogLogger
	skipLevel int
}

func (s SlogLogger) Warnf(format string, args ...interface{}) {
	if !s.Logger.Enabled(todo, slog.LevelWarn) {
		return
	}
	s.handle(slog.NewRecord(time.Now(), slog.LevelWarn, "", CallerPC()), format, args...)

}

func (s SlogLogger) GetSlogLogger() *slog.Logger {
	return s.Logger
}

func (s SlogLogger) Infof(format string, args ...interface{}) {
	if !s.Logger.Enabled(todo, slog.LevelInfo) {
		return
	}
	s.handle(slog.NewRecord(time.Now(), slog.LevelInfo, "", CallerPC()), format, args...)
}

func (s SlogLogger) Secretf(format string, args ...interface{}) {
	s.Debugf(StripSecrets(fmt.Sprintf(format, args...)))
}

func (s SlogLogger) Prettyf(msg string, obj interface{}) {
	pretty.Print(obj)
}

func (s SlogLogger) Errorf(format string, args ...interface{}) {
	if !s.Logger.Enabled(todo, slog.LevelError) {
		return
	}
	s.handle(slog.NewRecord(time.Now(), slog.LevelError, "", CallerPC()), format, args...)
}

func (s SlogLogger) Debugf(format string, args ...interface{}) {
	if !s.Logger.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	s.handle(slog.NewRecord(time.Now(), slog.LevelDebug, "", CallerPC()), format, args...)

}

func (s SlogLogger) handle(r slog.Record, format string, args ...interface{}) {
	if jsonLogs {
		if s.Prefix != "" {
			r.Add("logger", s.Prefix)
		}
		r.Message = fmt.Sprintf(format, args...)
	} else if s.Prefix != "" {
		r.Message = fmt.Sprintf(fmt.Sprintf("(%s) ", BrightF(s.Prefix))+format, args...)
	} else {
		r.Message = fmt.Sprintf(format, args...)
	}
	_ = s.Logger.Handler().Handle(context.Background(), r)
}

func (s SlogLogger) Tracef(format string, args ...interface{}) {
	if !s.Logger.Enabled(todo, SlogTraceLevel) {
		return
	}
	s.handle(slog.NewRecord(time.Now(), SlogTraceLevel, "", CallerPC()), format, args...)

}

func (s SlogLogger) Fatalf(format string, args ...interface{}) {
	s.handle(slog.NewRecord(time.Now(), SlogFatal, "", CallerPC()), format, args...)
}

func (s SlogLogger) DebugLevels() {
	s.Debugf("name=%s level=%d json=%v color=%v ", s.Prefix, s.GetLevel(), jsonLogs, color)
}

type slogVerbose struct {
	SlogLogger
	level slog.Level
}

func (v slogVerbose) Infof(format string, args ...interface{}) {
	if !v.Logger.Enabled(todo, v.level) {
		return
	}

	v.handle(slog.NewRecord(time.Now(), v.level, "", CallerPC()), format, args...)

}

func (v slogVerbose) Enabled() bool {
	return v.Logger.Enabled(context.Background(), v.level)
}

func (s SlogLogger) V(level any) Verbose {
	return &slogVerbose{
		SlogLogger: s,
		level:      ParseLevel(s, level).Slog(),
	}
}

func ParseLevel(logger Logger, level any) LogLevel {
	if lo.IsEmpty(level) {
		return Info
	}
	switch v := level.(type) {
	case slog.Level:
		return FromSlogLevel(v)
	case LogLevel:
		return v
	case int:
		return LogLevel(int(v))
	case string:

		// its a string e.g. "1"
		if i, err := strconv.Atoi(v); err == nil {
			return LogLevel(i)
		}

		v = strings.ToLower(v)
		// custom trace level e.g. trace7
		if strings.HasPrefix(v, "trace") {
			if i, err := strconv.Atoi(strings.TrimPrefix(v, "trace")); err == nil {
				return LogLevel(int(Trace) + i)
			}
		}
		switch v {
		case "debug":
			return Debug
		case "info":
			return Info
		case "warn":
			return Warn
		case "error":
			return Error
		case "fatal":
			return Fatal
		case "trace":
			return Trace
		default:
			if logger == nil {
				fmt.Printf("invalid log level: %v\n", level)
			} else {
				logger.Warnf("invalid log level: %v", level)
			}
		}
	default:
		return ParseLevel(logger, fmt.Sprintf("%v", level))
	}

	return Info
}

func (s SlogLogger) SetMinLogLevel(level any) {
	s.SetLogLevel(level)
}

func (s SlogLogger) IsLevelEnabled(level LogLevel) bool {
	return s.V(level).Enabled()
}

func FromSlogLevel(level slog.Level) LogLevel {
	switch level {
	case SlogTraceLevel:
		return Trace
	case slog.LevelDebug:
		return Debug
	case slog.LevelWarn:
		return Warn
	case slog.LevelInfo:
		return Info
	case slog.LevelError:
		return Error
	}
	return LogLevel(int(Trace) + (int(level)*-1 - int(SlogTraceLevel*-1)))
}

func (s SlogLogger) GetLevel() LogLevel {
	return FromSlogLevel(s.Level.Level())
}

func (level LogLevel) Slog() slog.Level {
	switch level {
	case Info:
		return slog.LevelInfo
	case Warn:
		return slog.LevelWarn
	case Error:
		return slog.LevelError
	case Trace:
		return SlogTraceLevel
	case Fatal:
		return SlogFatal
	}

	return slog.Level(int(SlogTraceLevel) - int(level-Trace))
}

func (s SlogLogger) WithV(level any) Logger {
	newlogger := s
	newlogger.Level = &slog.LevelVar{}
	newlogger.Level.Set(ParseLevel(s, level).Slog())
	return &newlogger
}

func (s SlogLogger) SetLogLevel(level any) {
	s.Level.Set(slog.Level(ParseLevel(s, level).Slog()))
}

func (s SlogLogger) Named(name string) Logger {
	return GetLogger(name)
}

func (s SlogLogger) WithoutName() Logger {
	return GetLogger()
}

func (s SlogLogger) WithSkipReportLevel(i int) Logger {
	return SlogLogger{
		Logger:    s.Logger,
		Level:     s.Level,
		Prefix:    s.Prefix,
		skipLevel: i,
	}
}

func (s SlogLogger) WithValues(keysAndValues ...interface{}) Logger {
	return SlogLogger{
		Logger: s.Logger.With(keysAndValues...),
		Level:  s.Level,
		Prefix: s.Prefix,
	}
}

func (s SlogLogger) IsTraceEnabled() bool {
	return s.Logger.Enabled(context.Background(), SlogTraceLevel)
}

func (s SlogLogger) IsDebugEnabled() bool {
	return s.Logger.Enabled(context.Background(), slog.LevelDebug)
}

func Pretty(v any) string {
	b, _ := json.MarshalIndent(v, "  ", "  ")
	return strings.TrimSpace(string(b))
}
