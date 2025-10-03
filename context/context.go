// Package context provides an enhanced context implementation that combines
// standard Go context functionality with integrated logging and distributed tracing.
//
// The Context type embeds context.Context and adds:
//   - Integrated structured logging with level control
//   - OpenTelemetry tracing support
//   - Debug and trace mode flags
//   - Automatic correlation between logs and traces
//
// Basic Usage:
//
//	ctx := context.NewContext(context.Background())
//	ctx.Infof("Processing request %s", requestID)
//
//	// Enable debug logging
//	debugCtx := ctx.WithDebug()
//	debugCtx.Debugf("Detailed information: %v", details)
//
//	// Start a traced operation
//	ctx, span := ctx.StartSpan("database-query")
//	defer span.End()
//	ctx.Infof("Executing query: %s", query)
//
// With Custom Logger and Tracer:
//
//	ctx := context.NewContext(
//		context.Background(),
//		context.WithLogger(customLogger),
//		context.WithTracer(otel.Tracer("my-service")),
//	)
//
// The context automatically propagates logging configuration and trace spans
// through the call chain, ensuring consistent observability across your application.
package context

import (
	gocontext "context"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	noopTracer = noop.NewTracerProvider().Tracer("noop")
)

// ContextOptions is a function that configures a Context during creation.
type ContextOptions func(*Context)

// WithTraceFn sets a custom function to determine if trace logging is enabled.
// This allows dynamic control of trace logging based on context values or external conditions.
//
// Example:
//
//	ctx := NewContext(context.Background(), WithTraceFn(func(ctx Context) *bool {
//		// Enable trace for specific user IDs
//		userID := ctx.Value("userID")
//		enabled := userID == "debug-user"
//		return &enabled
//	}))
func WithTraceFn(fn func(Context) *bool) ContextOptions {
	return func(opts *Context) {
		opts.isTraceFn = fn
	}
}

// WithDebugFn sets a custom function to determine if debug logging is enabled.
// This allows dynamic control of debug logging based on context values or external conditions.
func WithDebugFn(fn func(Context) *bool) ContextOptions {
	return func(opts *Context) {
		opts.isDebugFn = fn
	}
}

// WithTracer configures the OpenTelemetry tracer for the context.
// If not specified, a no-op tracer is used.
//
// Example:
//
//	tracer := otel.Tracer("my-service")
//	ctx := NewContext(context.Background(), WithTracer(tracer))
func WithTracer(tracer trace.Tracer) ContextOptions {
	return func(opts *Context) {
		opts.tracer = tracer
	}
}

// WithLogger sets a custom logger for the context.
// If not specified, the standard logger is used.
func WithLogger(log logger.Logger) ContextOptions {
	return func(opts *Context) {
		opts.Logger = log
	}
}

// NewContext creates a new enhanced context from a standard Go context.
// The context is configured with the provided options and defaults to
// the standard logger and a no-op tracer if not specified.
//
// Example:
//
//	// Basic context with defaults
//	ctx := NewContext(context.Background())
//
//	// Context with custom configuration
//	ctx := NewContext(
//		context.Background(),
//		WithLogger(myLogger),
//		WithTracer(myTracer),
//		WithDebugFn(customDebugCheck),
//	)
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

	if ctx.tracer == nil {
		ctx.tracer = noopTracer
	}
	return ctx
}

// Context is an enhanced context that embeds the standard context.Context
// and adds integrated logging and tracing capabilities.
//
// It provides:
//   - Structured logging with automatic trace correlation
//   - Debug and trace mode management
//   - OpenTelemetry span creation and management
//   - Context value propagation with type safety
//
// The Context maintains its configuration (logger, tracer, debug/trace settings)
// when creating derived contexts through WithValue, WithTimeout, etc.
type Context struct {
	gocontext.Context
	Logger    logger.Logger
	isDebugFn func(Context) *bool
	isTraceFn func(Context) *bool
	tracer    trace.Tracer
}

func (c Context) String() string {
	s := []string{}
	if c.IsTrace() {
		s = append(s, "[trace]")
	} else if c.IsDebug() {
		s = append(s, "[debug]")
	}
	if c.tracer != nil && c.tracer != noopTracer {
		s = append(s, "[otel]")
	}

	if c.isDebugFn != nil {
		s = append(s, fmt.Sprintf("isDebugFn=%v", lo.FromPtr(c.isDebugFn(c))))
	}
	if c.isTraceFn != nil {
		s = append(s, fmt.Sprintf("isTraceFn=%v", lo.FromPtr(c.isTraceFn(c))))
	}

	s = append(s, fmt.Sprintf("debug=%v, trace=%v", c.Value("debug"), c.Value("trace")))

	s = append(s, fmt.Sprintf("level=%d global=%d", c.Logger.GetLevel(), logger.StandardLogger().GetLevel()))

	return strings.Join(s, " ")
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
	ctx := c.WithValue("debug", "true")
	ctx.Logger = ctx.Logger.WithV(logger.Debug)
	return ctx
}

func (c Context) WithTrace() Context {
	ctx := c.WithValue("trace", "true")
	ctx.Logger = ctx.Logger.WithV(logger.Trace)
	return ctx
}

func (c Context) Clone() Context {
	return Context{
		Context:   c.Context,
		isDebugFn: c.isDebugFn,
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
	if c.IsTrace() {
		return true
	}

	if debug := c.Value("debug"); !lo.IsEmpty(debug) {
		return debug == "true"
	}

	if c.isDebugFn != nil {
		debug := c.isDebugFn(c)
		if debug != nil {
			return *debug || c.Logger.IsLevelEnabled(5)
		}
	}
	return c.Logger.IsDebugEnabled()
}

func (c Context) IsTrace() bool {
	if trace := c.Value("trace"); !lo.IsEmpty(trace) {
		return trace == "true"
	}
	if c.isTraceFn != nil {
		trace := c.isTraceFn(c)
		if trace != nil {
			return *trace || c.Logger.IsLevelEnabled(6)
		}
	}
	return c.Logger.IsTraceEnabled()
}

func (c Context) Debugf(format string, args ...interface{}) {
	if c.IsDebug() {
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "debug")))
		c.Logger.WithSkipReportLevel(1).Debugf(format, args...)
	}
}

func (c Context) Tracef(format string, args ...interface{}) {
	if c.IsTrace() {
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "trace")))
		c.Logger.WithSkipReportLevel(1).Tracef(format, args...)
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

	if o, ok := oops.AsOops(err); ok {
		c.Logger.WithSkipReportLevel(1).Errorf("%#v", o.ToMap())
	} else {
		c.Logger.WithSkipReportLevel(1).Errorf(err.Error())
	}
}

func (c Context) Errorf(format string, args ...interface{}) {
	err := fmt.Sprintf(format, args...)
	c.GetSpan().RecordError(errors.New(err))
	c.GetSpan().SetStatus(codes.Error, err)
	c.Logger.WithSkipReportLevel(1).Errorf(err)
}

func (c Context) Infof(format string, args ...interface{}) {
	if c.IsDebug() {
		// info level logs should only be pushed for debug traces
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "info")))
	}
	c.Logger.WithSkipReportLevel(1).Infof(fmt.Sprintf(format, args...))
}

func (c Context) Warnf(format string, args ...interface{}) {
	if c.IsDebug() {
		// info level logs should only be pushed for debug traces
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", "warn")))
	}
	c.Logger.WithSkipReportLevel(1).Warnf(fmt.Sprintf(format, args...))
}

func (c Context) Logf(level int, format string, args ...interface{}) {
	if c.IsTrace() {
		// info level logs should only be pushed for debug traces
		c.GetSpan().AddEvent(fmt.Sprintf(format, args...), trace.WithAttributes(attribute.String("level", fmt.Sprintf("%d", level))))
	}
	c.Logger.WithSkipReportLevel(1).V(level).Infof(format, args...)
}

func (c Context) GetSpan() trace.Span {
	return trace.SpanFromContext(c)
}

func (c Context) WithoutSpan() Context {
	return Context{
		Context:   trace.ContextWithSpanContext(c.Context, c.GetSpan().SpanContext()),
		Logger:    c.Logger,
		isDebugFn: c.isDebugFn,
		isTraceFn: c.isTraceFn,
	}
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
