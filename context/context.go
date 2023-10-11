package context

import (
	gocontext "context"
	"fmt"

	"github.com/flanksource/commons/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ContextOptions func(*Context)

func WithTraceFn(fn func(*Context) bool) ContextOptions {
	return func(opts *Context) {
		opts.isTraceFn = fn
	}
}

func WithDebugFn(fn func(*Context) bool) ContextOptions {
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

func NewContext(opts ...ContextOptions) *Context {
	ctx := Context{
		Context: gocontext.Background(),
	}
	for _, opt := range opts {
		opt(&ctx)
	}
	if ctx.logger == nil {
		ctx.logger = logger.StandardLogger()
	}
	if ctx.isDebugFn == nil {
		ctx.isDebugFn = func(*Context) bool {
			return ctx.logger.IsDebugEnabled()
		}
	}
	if ctx.isTraceFn == nil {
		ctx.isTraceFn = func(*Context) bool {
			return ctx.logger.IsTraceEnabled()
		}
	}
	if ctx.tracer == nil {
		ctx.tracer = trace.NewNoopTracerProvider().Tracer("noop")
	}
	return &ctx
}

type Context struct {
	gocontext.Context
	logger    logger.Logger
	isDebugFn func(*Context) bool
	isTraceFn func(*Context) bool
	tracer    trace.Tracer
}

func (c *Context) WithTracer(tracer trace.Tracer) {
	c.tracer = tracer
}

func (c *Context) WithValue(key, val interface{}) *Context {
	return &Context{
		Context:   gocontext.WithValue(c, key, val),
		isDebugFn: c.isDebugFn,
		logger:    c.logger,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}
}

func (c *Context) IsDebug() bool {
	return c.isDebugFn(c)
}

func (c *Context) IsTrace() bool {
	return c.isTraceFn(c)
}

func (c *Context) Debugf(format string, args ...interface{}) {
	if c.isDebugFn(c) {
		if c.HasSpanStarted() {
			c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "debug")))
		}
		c.logger.Debugf(format, args...)
	}
}

func (c *Context) Tracef(format string, args ...interface{}) {
	if c.isTraceFn(c) {
		if c.HasSpanStarted() {
			c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "trace")))
		}
		c.logger.Tracef(format, args...)
	}
}

func (c *Context) Error(err error) {
	c.GetSpan().RecordError(err)
	c.GetSpan().SetStatus(codes.Error, "")
}

func (c *Context) Errorf(err error, format string, args ...interface{}) {
	c.GetSpan().RecordError(err)
	c.GetSpan().SetStatus(codes.Error, fmt.Sprintf(format, args...))
}

func (c *Context) HasSpanStarted() bool {
	span := c.Value(0)
	return span != nil
}

func (c *Context) GetSpan() trace.Span {
	return trace.SpanFromContext(c)
}

func (c *Context) StartSpan(name string) (*Context, trace.Span) {
	ctx, span := c.tracer.Start(c, name)
	return &Context{
		Context:   ctx,
		logger:    c.logger,
		isDebugFn: c.isDebugFn,
		isTraceFn: c.isTraceFn,
		tracer:    c.tracer,
	}, span
}
