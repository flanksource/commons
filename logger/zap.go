package logger

import (
	"fmt"
	"os"

	"github.com/kr/pretty"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	zapapi "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	Json            bool
	Level           *zapapi.AtomicLevel
	Base            *zapapi.Logger
	Logger          *zapapi.SugaredLogger
	LevelEncoder    zapcore.LevelEncoder
	TimeEncoder     zapcore.TimeEncoder
	StackTraceLevel *zap.AtomicLevel
}

type zapVerbose struct {
	*ZapLogger
	level int
}

func GetZapLogger() *ZapLogger {
	switch v := currentLogger.(type) {
	case ZapLogger:
		zapLogger := v
		return &zapLogger
	}
	return nil
}

func (logger ZapLogger) GetLevel() *zapapi.AtomicLevel {
	return logger.Level
}
func (logger ZapLogger) GetEncoder() zapcore.Encoder {
	if logger.Json {
		return zapcore.NewJSONEncoder(logger.GetEncoderConfig())
	}
	return zapcore.NewConsoleEncoder(logger.GetEncoderConfig())
}

func (logger ZapLogger) GetEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		TimeKey:        "timestamp",
		EncodeLevel:    logger.LevelEncoder,
		EncodeTime:     logger.TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
}

func UseZap(flags *pflag.FlagSet) {
	level, _ := flags.GetCount("loglevel")
	json, _ := flags.GetBool("json-logs")
	logger := ZapLogger{Json: json}
	logger.LevelEncoder = zapcore.CapitalColorLevelEncoder
	if json {
		logger.LevelEncoder = zapcore.LowercaseLevelEncoder
	}
	logger.TimeEncoder = zapcore.ISO8601TimeEncoder
	atom := zapapi.NewAtomicLevelAt(zapcore.InfoLevel - zapcore.Level(level))
	logger.Level = &atom
	stacktraceLevel := zap.NewAtomicLevelAt(zap.ErrorLevel)
	logger.StackTraceLevel = &stacktraceLevel
	core := zapcore.NewCore(logger.GetEncoder(), os.Stdout, atom.Level())
	logger.Base = zap.New(core).WithOptions(zapapi.AddStacktrace(logger.StackTraceLevel), zap.AddCallerSkip(1))
	logger.Logger = logger.Base.Sugar()
	currentLogger = logger
}

func (zap ZapLogger) Warnf(format string, args ...interface{}) {
	zap.Logger.Warnf(format, args...)
}

func (zap ZapLogger) Infof(format string, args ...interface{}) {
	zap.Logger.Infof(format, args...)
}

func (zap ZapLogger) Secretf(format string, args ...interface{}) {
	zap.Tracef(stripSecrets(fmt.Sprintf(format, args...)))
}

func (zap ZapLogger) Prettyf(msg string, obj interface{}) {
	pretty.Print(obj)
}

func (zap ZapLogger) Errorf(format string, args ...interface{}) {
	zap.Logger.Errorf(format, args...)
}

func (zap ZapLogger) Debugf(format string, args ...interface{}) {
	zap.Logger.Debugf(format, args...)
}

func (zap ZapLogger) Tracef(format string, args ...interface{}) {
	zap.Logger.Debugf(format, args...)
}

func (zap ZapLogger) Fatalf(format string, args ...interface{}) {
	zap.Logger.Fatalf(format, args...)
}

func (v zapVerbose) Info(args ...interface{}) {
	switch v.level {
	case 0:
		v.Logger.Info(args...)

	default:
		v.Logger.Debug(args...)
	}
}

func (v zapVerbose) Infof(format string, args ...interface{}) {
	switch v.level {
	case 0:
		v.Logger.Infof(format, args...)

	default:
		v.Logger.Debugf(format, args...)
	}
}

func (v zapVerbose) Infoln(args ...interface{}) {
	switch v.level {
	case 0:
		v.Logger.Info(args...)

	default:
		v.Logger.Info(args...)
	}
}

func (zap ZapLogger) V(level int) Verbose {
	return &zapVerbose{
		ZapLogger: &zap,
		level:     level,
	}
}

func (zap ZapLogger) SetLogLevel(level int) {
	atom := zapapi.NewAtomicLevelAt(zapcore.InfoLevel - zapcore.Level(level))
	zap.Level = &atom
	zap.Level.SetLevel(atom.Level())
}

func (zap ZapLogger) WithValues(keysAndValues ...interface{}) Logger {
	logger := zap.Logger.With(keysAndValues...)
	return ZapLogger{
		Level:  zap.Level,
		Base:   logger.Desugar(),
		Logger: logger,
	}
}

func (zap ZapLogger) IsTraceEnabled() bool {
	return zap.Base.Core().Enabled(zapcore.DebugLevel - zapcore.Level(1))
}

func (zap ZapLogger) IsDebugEnabled() bool {
	return zap.Base.Core().Enabled(zapcore.DebugLevel)
}
