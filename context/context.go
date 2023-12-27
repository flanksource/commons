package context

import (
	gocontext "context"
	"fmt"
	"time"

	"github.com/flanksource/commons/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	noopTracer = trace.NewNoopTracerProvider().Tracer("noop")
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
		opts.logger = log
	}
}

func NewContext(basectx gocontext.Context, opts ...ContextOptions) Context {
	ctx := Context{
		Context: basectx,
	}
	for _, opt := range opts {
		opt(&ctx)
	}
	if ctx.logger == nil {
		ctx.logger = logger.StandardLogger()
	}
	if ctx.isDebugFn == nil {
		ctx.isDebugFn = func(Context) bool {
			return ctx.logger.IsDebugEnabled()
		}
	}
	if ctx.isTraceFn == nil {
		ctx.isTraceFn = func(Context) bool {
			return ctx.logger.IsTraceEnabled()
		}
	}
	if ctx.tracer == nil {
		ctx.tracer = noopTracer
	}
	return ctx
}

type Context struct {
	gocontext.Context
	logger    logger.Logger
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
		logger:    c.logger,
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
	c.logger.SetLogLevel(1)
	return c
}

func (c Context) WithTrace() Context {
	t := true
	c.trace = &t
	c.logger.SetLogLevel(2)
	return c
}

func (c Context) Clone() Context {
	return Context{
		Context:   c.Context,
		isDebugFn: c.isDebugFn,
		trace:     c.trace,
		debug:     c.debug,
		logger:    c.logger,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}
}

func (c Context) WithTimeout(timeout time.Duration) (Context, gocontext.CancelFunc) {
	ctx, cancelFunc := gocontext.WithTimeout(c, timeout)
	return Context{
		Context:   ctx,
		isDebugFn: c.isDebugFn,
		logger:    c.logger,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}, cancelFunc
}

func (c Context) IsDebug() bool {
	return (c.debug != nil && *c.debug) || c.isDebugFn(c)
}

func (c Context) IsTrace() bool {
	return (c.trace != nil && *c.trace) || c.isTraceFn(c)
}

func (c Context) Debugf(format string, args ...interface{}) {
	if c.IsDebug() {
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "debug")))
		c.logger.Debugf(format, args...)
	}
}

func (c Context) Tracef(format string, args ...interface{}) {
	if c.IsTrace() {
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "trace")))
		c.logger.Tracef(format, args...)
	}
}

func (c Context) Error(err error) {
	c.GetSpan().RecordError(err)
	c.GetSpan().SetStatus(codes.Error, err.Error())
	c.logger.Errorf(err.Error())
}

func (c Context) Errorf(err error, format string, args ...interface{}) {
	c.GetSpan().RecordError(err)
	c.GetSpan().SetStatus(codes.Error, fmt.Sprintf(format, args...))
	c.logger.Errorf(fmt.Sprintf(format, args...))
}

func (c Context) GetSpan() trace.Span {
	return trace.SpanFromContext(c)
}

func (c Context) StartSpan(name string) (Context, trace.Span) {
	ctx, span := c.tracer.Start(c, name)
	return Context{
		Context:   ctx,
		logger:    c.logger,
		isDebugFn: c.isDebugFn,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}, span
}
