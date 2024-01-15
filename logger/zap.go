package logger

import (
	"fmt"
	"os"

	"github.com/kr/pretty"
	"go.uber.org/zap"
	zapapi "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	cyan    = "\x1b[36"
	Cyan    = cyan + Normal
	magenta = "\x1b[35"
	Magenta = magenta + Normal
	Normal  = "m"
	Reset   = "\x1b[0m"
)
const TraceLevel = zapcore.DebugLevel - 1

type ZapLogger struct {
	*zapapi.Logger
	Sugar       *zapapi.SugaredLogger
	atomicLevel *zapapi.AtomicLevel
}

type zapVerbose struct {
	*ZapLogger
	level zapcore.Level
}

func GetZapLogger() *ZapLogger {
	switch v := currentLogger.(type) {
	case ZapLogger:
		zapLogger := v
		return &zapLogger
	}
	return nil
}

func (z ZapLogger) IsLevelEnabled(level int) bool {
	return z.Logger.Core().Enabled(zap.InfoLevel - zapcore.Level(level))
}

func (z ZapLogger) GetLevel() int {
	return int(z.Level()) * -1
}

func (z ZapLogger) Named(name string) Logger {
	logger := z.Logger.Named(name)
	var level = *z.atomicLevel
	return ZapLogger{
		Sugar:       logger.Sugar(),
		Logger:      logger,
		atomicLevel: &level,
	}
}

func UseZap() {
	currentLogger = newZap(level)
}

func newZap(level int) ZapLogger {
	var encoder zapcore.Encoder

	capitalColorLevelEncoder := func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		if level <= TraceLevel {
			enc.AppendString(Magenta + "TRACE" + Reset)
			return
		}
		zapcore.CapitalColorLevelEncoder(level, enc)
	}

	if jsonLogs {
		config := zap.NewProductionEncoderConfig()
		config.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			if level <= TraceLevel {
				enc.AppendString("trace")
				return
			}
			zapcore.LowercaseLevelEncoder(level, enc)
		}
		config.EncodeTime = zapcore.RFC3339NanoTimeEncoder
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000")
		config.EncodeLevel = capitalColorLevelEncoder
		config.EncodeCaller = zapcore.ShortCallerEncoder

		encoder = zapcore.NewConsoleEncoder(config)

	}

	atomicLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel - zapcore.Level(level))
	zapCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), atomicLevel)
	var opts []zapapi.Option
	if reportCaller {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(2))
	}
	logger := zap.New(zapCore, opts...)
	return ZapLogger{
		Logger:      logger,
		Sugar:       logger.Sugar(),
		atomicLevel: &atomicLevel,
	}
}

func (zap ZapLogger) Warnf(format string, args ...interface{}) {
	zap.Sugar.Warnf(format, args...)
}

func (zap ZapLogger) Infof(format string, args ...interface{}) {
	zap.Sugar.Infof(format, args...)
}

func (zap ZapLogger) Secretf(format string, args ...interface{}) {
	zap.Tracef(stripSecrets(fmt.Sprintf(format, args...)))
}

func (zap ZapLogger) Prettyf(msg string, obj interface{}) {
	pretty.Print(obj)
}

func (zap ZapLogger) Errorf(format string, args ...interface{}) {
	zap.Sugar.Errorf(format, args...)
}

func (zap ZapLogger) Debugf(format string, args ...interface{}) {
	zap.Sugar.Debugf(format, args...)
}

func (zap ZapLogger) Tracef(format string, args ...interface{}) {
	zap.Log(TraceLevel, fmt.Sprintf(format, args...))
}

func (zap ZapLogger) Fatalf(format string, args ...interface{}) {
	zap.Sugar.Fatalf(format, args...)
}

func (v zapVerbose) Infof(format string, args ...interface{}) {
	v.Log(v.level, fmt.Sprintf(format, args...))
}

func (v zapVerbose) Enabled() bool {
	return v.Logger.Level().Enabled(v.level)
}

func (zap ZapLogger) V(level int) Verbose {
	return &zapVerbose{
		ZapLogger: &zap,
		level:     zapcore.InfoLevel - zapcore.Level(level),
	}
}

func (zap ZapLogger) SetMinLogLevel(level int) {
	if zap.GetLevel() >= level {
		return
	}
	zap.atomicLevel.SetLevel(zapcore.InfoLevel - zapcore.Level(level))
}

func (zap ZapLogger) SetLogLevel(level int) {
	zap.atomicLevel.SetLevel(zapcore.InfoLevel - zapcore.Level(level))
}

func (z ZapLogger) WithEncoder(encoder zapcore.Encoder) ZapLogger {
	level := *z.atomicLevel
	zapCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), level)
	logger := zap.New(zapCore)
	return ZapLogger{
		Logger:      logger,
		Sugar:       logger.Sugar(),
		atomicLevel: &level,
	}
}

func (zap ZapLogger) WithValues(keysAndValues ...interface{}) Logger {
	logger := zap.Sugar.With(keysAndValues...)
	var level = *zap.atomicLevel
	return ZapLogger{
		Sugar:       logger,
		Logger:      logger.Desugar(),
		atomicLevel: &level,
	}
}

func (zap ZapLogger) IsTraceEnabled() bool {
	return zap.Logger.Core().Enabled(TraceLevel)
}

func (zap ZapLogger) IsDebugEnabled() bool {
	return zap.Logger.Core().Enabled(zapcore.DebugLevel)
}
