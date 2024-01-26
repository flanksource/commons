package context

import (
	gocontext "context"
	"fmt"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	noopTracer = noop.NewTracerProvider().Tracer("noop")
)

type ContextOptions func(*Context)

func WithTraceFn(fn func(Context) bool) ContextOptions {
	return func(opts *Context) {
		opts.isTraceFn = fn
	}
}

func WithDebugFn(fn func(Context) bool) ContextOptions {
	return func(opts *Context) {
		opts.isDebugFn = fn
	}
}

func WithTracer(tracer trace.Tracer) ContextOptions {
	return func(opts *Context) {
		opts.tracer = tracer
	}
}

func WithLogger(log logger.Logger) ContextOptions {
	return func(opts *Context) {
		opts.Logger = log
	}
}

func NewContext(basectx gocontext.Context, opts ...ContextOptions) Context {
	ctx := Context{
		Context: basectx,
	}
	for _, opt := range opts {
		opt(&ctx)
	}
	if ctx.Logger == nil {
		ctx.Logger = logger.StandardLogger()
	}
	if ctx.isDebugFn == nil {
		ctx.isDebugFn = func(Context) bool {
			return ctx.Logger.IsDebugEnabled()
		}
	}
	if ctx.isTraceFn == nil {
		ctx.isTraceFn = func(Context) bool {
			return ctx.Logger.IsTraceEnabled()
		}
	}
	if ctx.tracer == nil {
		ctx.tracer = noopTracer
	}
	return ctx
}

type Context struct {
	gocontext.Context
	Logger    logger.Logger
	debug     *bool
	trace     *bool
	isDebugFn func(Context) bool
	isTraceFn func(Context) bool
	tracer    trace.Tracer
}

func (c *Context) WithTracer(tracer trace.Tracer) {
	c.tracer = tracer
}

func (c Context) WithValue(key, val interface{}) Context {
	return Context{
		Context:   gocontext.WithValue(c, key, val),
		isDebugFn: c.isDebugFn,
		Logger:    c.Logger,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}
}

func (c Context) GetTracer() trace.Tracer {
	if c.tracer == nil {
		return noopTracer
	}
	return c.tracer
}

func (c Context) WithDebug() Context {
	t := true
	c.debug = &t
	c.Logger.SetMinLogLevel(1)
	return c
}

func (c Context) WithTrace() Context {
	t := true
	c.trace = &t
	c.Logger.SetMinLogLevel(2)
	return c
}

func (c Context) Clone() Context {
	return Context{
		Context:   c.Context,
		isDebugFn: c.isDebugFn,
		trace:     c.trace,
		debug:     c.debug,
		Logger:    c.Logger,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}
}

func (c Context) WithTimeout(timeout time.Duration) (Context, gocontext.CancelFunc) {
	ctx, cancelFunc := gocontext.WithTimeout(c, timeout)
	return Context{
		Context:   ctx,
		isDebugFn: c.isDebugFn,
		Logger:    c.Logger,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}, cancelFunc
}

func (c Context) WithDeadline(deadline time.Time) (Context, gocontext.CancelFunc) {
	ctx, cancelFunc := gocontext.WithDeadline(c, deadline)
	return Context{
		Context:   ctx,
		isDebugFn: c.isDebugFn,
		Logger:    c.Logger,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}, cancelFunc
}

func (c Context) IsDebug() bool {
	return c.IsTrace() || c.Logger.IsLevelEnabled(5) ||
		(c.debug != nil && *c.debug) ||
		(c.isDebugFn != nil && c.isDebugFn(c))
}

func (c Context) IsTrace() bool {
	return c.Logger.IsLevelEnabled(6) || (c.trace != nil && *c.trace) || (c.isTraceFn != nil && c.isTraceFn(c))
}

func (c Context) Debugf(format string, args ...interface{}) {
	if c.IsDebug() {
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "debug")))
		c.Logger.Debugf(format, args...)
	}
}

func (c Context) Tracef(format string, args ...interface{}) {
	if c.IsTrace() {
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "trace")))
		c.Logger.Tracef(format, args...)
	}
}

func (c Context) Error(err error, msg ...any) {
	if len(msg) == 1 {
		err = errors.Wrap(err, fmt.Sprintf("%v", msg[0]))
	} else if len(msg) > 1 {
		err = errors.Wrap(err, fmt.Sprintf(fmt.Sprintf("%v", msg[0]), lo.ToAnySlice(msg[1:])...))
	}

	c.GetSpan().RecordError(err)
	c.GetSpan().SetStatus(codes.Error, err.Error())
	c.Logger.Errorf(err.Error())
}

func (c Context) Errorf(format string, args ...interface{}) {
	err := fmt.Sprintf(format, args...)
	c.GetSpan().RecordError(errors.New(err))
	c.GetSpan().SetStatus(codes.Error, err)
	c.Logger.Errorf(err)
}

func (c Context) Infof(format string, args ...interface{}) {
	if c.IsDebug() {
		// info level logs should only be pushed for debug traces
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "info")))
	}
	c.Logger.Infof(fmt.Sprintf(format, args...))
}

func (c Context) Warnf(format string, args ...interface{}) {
	if c.IsDebug() {
		// info level logs should only be pushed for debug traces
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "warn")))
	}
	c.Logger.Warnf(fmt.Sprintf(format, args...))
}

func (c Context) Logf(level int, format string, args ...interface{}) {
	if c.IsTrace() {
		// info level logs should only be pushed for debug traces
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", fmt.Sprintf("%d", level))))
	}
	c.Logger.V(level).Infof(format, args...)
}

func (c Context) GetSpan() trace.Span {
	return trace.SpanFromContext(c)
}

func (c Context) StartSpan(name string) (Context, trace.Span) {
	ctx, span := c.tracer.Start(c, name)
	return Context{
		Context:   ctx,
		Logger:    c.Logger,
		isDebugFn: c.isDebugFn,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}, span
}
