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

	"github.com/flanksource/commons/is"
	"github.com/flanksource/commons/properties"
	"github.com/kr/pretty"
	"github.com/lmittmann/tint"
	"github.com/lrita/cmap"
)

var (
	isTTY = is.TTY()
)

var namedLoggers cmap.Map[string, *SlogLogger]

func BrightF(msg string, args ...interface{}) string {
	if !color || isTTY && !jsonLogs {
		return DarkWhite + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

var SlogTraceLevel slog.Level = slog.LevelDebug - 1
var SlogTraceLevel1 slog.Level = SlogTraceLevel - 1
var SlogTraceLevel2 slog.Level = SlogTraceLevel - 2
var SlogTraceLevel3 slog.Level = SlogTraceLevel - 3
var SlogTraceLevel4 slog.Level = SlogTraceLevel - 4
var SlogFatal = slog.LevelError + 1

func GetSlogLogger() SlogLogger {
	return currentLogger.(SlogLogger)
}

func onPropertyUpdate(props *properties.Properties) {
	for k, v := range props.GetAll() {

		if k == "log.level" || k == "log.json" || k == "log.caller" || k == "log.color" {
			root := New("root")
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
}

func New(prefix string) *SlogLogger {
	// create a new slogger
	var slogger *slog.Logger
	var lvl = &slog.LevelVar{}
	var l any

	reportCaller := properties.On(false, fmt.Sprintf("log.caller.%s", prefix), "log.caller")
	logJson := properties.On(false, fmt.Sprintf("log.json.%s", prefix), "log.json")
	logColor := properties.On(false, fmt.Sprintf("log.color.%s", prefix), "log.color")
	logLevel := properties.String("", fmt.Sprintf("log.level.%s", prefix), "log.level")
	if logJson {
		slogger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: reportCaller,
			Level:     lvl,
		}))
	} else {
		slogger = slog.New(tint.NewHandler(os.Stderr, &tint.Options{
			Level:      lvl,
			NoColor:    !logColor,
			AddSource:  reportCaller,
			TimeFormat: properties.String("15:04:05.999", fmt.Sprintf("log.time.format.%s", prefix), "log.time.format"),
		}))
	}
	logger := SlogLogger{
		Logger: slogger,
		Level:  lvl,
	}

	if logLevel != "" {
		l = logLevel
	} else {
		l = level
	}

	if prefix != "" {
		logger.Prefix = fmt.Sprintf("[%s] ", BrightF(prefix))
	}
	logger.SetLogLevel(l)
	return &logger
}
func UseSlog() {
	if currentLogger != nil {
		return
	}

	root := New("root")

	slog.SetDefault(root.Logger)
	namedLoggers.Store("root", root)
	currentLogger = root

	properties.RegisterListener(onPropertyUpdate)
}

func GetLogger(names ...string) *SlogLogger {
	parent, _ := namedLoggers.Load("root")
	if len(names) == 0 {
		return parent
	}

	key := strings.ToLower(strings.Join(names, ""))
	if key == "" {
		return parent
	}

	if v, ok := namedLoggers.Load(key); ok {
		return v
	}
	child, _ := namedLoggers.LoadOrStore(key, New(key))
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
	r := slog.NewRecord(time.Now(), slog.LevelWarn, fmt.Sprintf(s.Prefix+format, args...), CallerPC())
	_ = s.Logger.Handler().Handle(context.Background(), r)
}

func (s SlogLogger) GetSlogLogger() *slog.Logger {
	return s.Logger
}

func (s SlogLogger) Infof(format string, args ...interface{}) {
	r := slog.NewRecord(time.Now(), slog.LevelInfo, fmt.Sprintf(s.Prefix+format, args...), CallerPC())
	_ = s.Logger.Handler().Handle(context.Background(), r)
}

func (s SlogLogger) Secretf(format string, args ...interface{}) {
	s.Debugf(stripSecrets(fmt.Sprintf(format, args...)))
}

func (s SlogLogger) Prettyf(msg string, obj interface{}) {
	pretty.Print(obj)
}

func (s SlogLogger) Errorf(format string, args ...interface{}) {
	r := slog.NewRecord(time.Now(), slog.LevelError, fmt.Sprintf(s.Prefix+format, args...), CallerPC())
	_ = s.Logger.Handler().Handle(context.Background(), r)
}

func (s SlogLogger) Debugf(format string, args ...interface{}) {
	r := slog.NewRecord(time.Now(), slog.LevelDebug, fmt.Sprintf(s.Prefix+format, args...), CallerPC())
	_ = s.Logger.Handler().Handle(context.Background(), r)
}

func (s SlogLogger) Tracef(format string, args ...interface{}) {
	r := slog.NewRecord(time.Now(), SlogTraceLevel, fmt.Sprintf(s.Prefix+format, args...), CallerPC())
	_ = s.Logger.Handler().Handle(context.Background(), r)
}

func (s SlogLogger) Fatalf(format string, args ...interface{}) {
	r := slog.NewRecord(time.Now(), SlogFatal, fmt.Sprintf(s.Prefix+format, args...), CallerPC())
	_ = s.Logger.Handler().Handle(context.Background(), r)
}

type slogVerbose struct {
	SlogLogger
	level slog.Level
}

func (v slogVerbose) Infof(format string, args ...interface{}) {
	r := slog.NewRecord(time.Now(), v.level, fmt.Sprintf(v.Prefix+format, args...), CallerPC())
	_ = v.Logger.Handler().Handle(context.Background(), r)
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
	switch v := level.(type) {
	case slog.Level:
		return FromSlogLevel(v)
	case LogLevel:
		return v
	case int:
		return LogLevel(int(v))
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return LogLevel(i)
		}
		switch strings.ToLower(v) {
		case "trace":
			return Trace
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
		case "trace1":
			return Trace1
		case "trace2":
			return Trace2
		case "trace3":
			return Trace3
		case "trace4":
			return Trace4
		default:
			logger.Warnf("invalid log level: %v", level)
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
	case SlogTraceLevel1:
		return Trace1
	case SlogTraceLevel2:
		return Trace2
	case SlogTraceLevel3:
		return Trace3
	case SlogTraceLevel4:
		return Trace4
	case SlogTraceLevel:
		return Trace
	case slog.LevelDebug:
		return Debug
	case slog.LevelWarn:
		return Warn
	case slog.LevelError:
		return Error
	}

	return Info
}
func (s SlogLogger) GetLevel() LogLevel {
	return FromSlogLevel(s.Level.Level())
}

func (level LogLevel) Slog() slog.Level {
	switch level {
	case Trace1:
		return SlogTraceLevel1
	case Trace2:
		return SlogTraceLevel2
	case Trace3:
		return SlogTraceLevel3
	case Trace4:
		return SlogTraceLevel4
	case Trace:
		return SlogTraceLevel
	case Debug:
		return slog.LevelDebug
	case Info:
		return slog.LevelInfo
	case Warn:
		return slog.LevelWarn
	case Error:
		return slog.LevelError
	case Fatal:
		return SlogFatal
	}

	return slog.LevelInfo
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
	b, _ := json.MarshalIndent(v, "", "  ")
	return strings.TrimSpace(string(b))
}
