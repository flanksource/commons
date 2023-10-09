package trace

import (
	gocontext "context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
)

type EventOption interface {
	applyEvent(EventConfig) EventConfig
}

// EventConfig is a group of options for an Event.
type EventConfig struct {
	attributes []attribute.KeyValue
	timestamp  time.Time
	stackTrace bool
}

// Attributes describe the associated qualities of an Event.
func (cfg *EventConfig) Attributes() []attribute.KeyValue {
	return cfg.attributes
}

// Timestamp is a time in an Event life-cycle.
func (cfg *EventConfig) Timestamp() time.Time {
	return cfg.timestamp
}

// StackTrace checks whether stack trace capturing is enabled.
func (cfg *EventConfig) StackTrace() bool {
	return cfg.stackTrace
}

// NewEventConfig applies all the EventOptions to a returned EventConfig. If no
// timestamp option is passed, the returned EventConfig will have a Timestamp
// set to the call time, otherwise no validation is performed on the returned
// EventConfig.
func NewEventConfig(options ...EventOption) EventConfig {
	var c EventConfig
	for _, option := range options {
		c = option.applyEvent(c)
	}
	if c.timestamp.IsZero() {
		c.timestamp = time.Now()
	}
	return c
}

// Span is the individual component of a trace. It represents a single named
// and timed operation of a workflow that is traced. A Tracer is used to
// create a Span and it is then up to the operation the Span represents to
// properly end the Span when the operation itself ends.
//
// Warning: methods may be added to this interface in minor releases.
type Span interface {
	// End completes the Span. The Span is considered complete and ready to be
	// delivered through the rest of the telemetry pipeline after this method
	// is called. Therefore, updates to the Span are not allowed after this
	// method has been called.
	End(options ...trace.SpanEndOption)

	// AddEvent adds an event with the provided name and options.
	AddEvent(name string, options ...EventOption)

	// IsRecording returns the recording state of the Span. It will return
	// true if the Span is active and events can be recorded.
	IsRecording() bool

	// RecordError will record err as an exception span event for this span. An
	// additional call to SetStatus is required if the Status of the Span should
	// be set to Error, as this method does not change the Span status. If this
	// span is not being recorded or err is nil then this method does nothing.
	RecordError(err error, options ...EventOption)

	// SpanContext returns the SpanContext of the Span. The returned SpanContext
	// is usable even after the End method has been called for the Span.
	SpanContext() trace.SpanContext

	// SetStatus sets the status of the Span in the form of a code and a
	// description, provided the status hasn't already been set to a higher
	// value before (OK > Error > Unset). The description is only included in a
	// status when the code is for an error.
	SetStatus(code codes.Code, description string)

	// SetName sets the Span name.
	SetName(name string)

	// SetAttributes sets kv as attributes of the Span. If a key from kv
	// already exists for an attribute of the Span it will be overwritten with
	// the value contained in kv.
	SetAttributes(kv ...attribute.KeyValue)

	// TracerProvider returns a TracerProvider that can be used to generate
	// additional Spans on the same telemetry pipeline as the current Span.
	TracerProvider() TracerProvider
}

// TracerProvider provides Tracers that are used by instrumentation code to
// trace computational workflows.
//
// A TracerProvider is the collection destination of all Spans from Tracers it
// provides, it represents a unique telemetry collection pipeline. How that
// pipeline is defined, meaning how those Spans are collected, processed, and
// where they are exported, depends on its implementation. Instrumentation
// authors do not need to define this implementation, rather just use the
// provided Tracers to instrument code.
//
// Commonly, instrumentation code will accept a TracerProvider implementation
// at runtime from its users or it can simply use the globally registered one
// (see https://pkg.go.dev/go.opentelemetry.io/otel#GetTracerProvider).
//
// Warning: methods may be added to this interface in minor releases.
type TracerProvider interface {
	// Tracer returns a unique Tracer scoped to be used by instrumentation code
	// to trace computational workflows. The scope and identity of that
	// instrumentation code is uniquely defined by the name and options passed.
	//
	// The passed name needs to uniquely identify instrumentation code.
	// Therefore, it is recommended that name is the Go package name of the
	// library providing instrumentation (note: not the code being
	// instrumented). Instrumentation libraries can have multiple versions,
	// therefore, the WithInstrumentationVersion option should be used to
	// distinguish these different codebases. Additionally, instrumentation
	// libraries may sometimes use traces to communicate different domains of
	// workflow data (i.e. using spans to communicate workflow events only). If
	// this is the case, the WithScopeAttributes option should be used to
	// uniquely identify Tracers that handle the different domains of workflow
	// data.
	//
	// If the same name and options are passed multiple times, the same Tracer
	// will be returned (it is up to the implementation if this will be the
	// same underlying instance of that Tracer or not). It is not necessary to
	// call this multiple times with the same name and options to get an
	// up-to-date Tracer. All implementations will ensure any TracerProvider
	// configuration changes are propagated to all provided Tracers.
	//
	// If name is empty, then an implementation defined default name will be
	// used instead.
	//
	// This method is safe to call concurrently.
	Tracer(name string, options ...TracerOption) Tracer
}

// Tracer is the creator of Spans.
//
// Warning: methods may be added to this interface in minor releases.
type Tracer interface {
	// Start creates a span and a context.Context containing the newly-created span.
	//
	// If the context.Context provided in `ctx` contains a Span then the newly-created
	// Span will be a child of that span, otherwise it will be a root span. This behavior
	// can be overridden by providing `WithNewRoot()` as a SpanOption, causing the
	// newly-created Span to be a root span even if `ctx` contains a Span.
	//
	// When creating a Span it is recommended to provide all known span attributes using
	// the `WithAttributes()` SpanOption as samplers will only have access to the
	// attributes provided when a Span is created.
	//
	// Any Span that is created MUST also be ended. This is the responsibility of the user.
	// Implementations of this API may leak memory or other resources if Spans are not ended.
	Start(ctx gocontext.Context, spanName string) (gocontext.Context, Span)
}

// TracerOption applies an option to a TracerConfig.
type TracerOption interface {
	apply(TracerConfig) TracerConfig
}

// TracerConfig is a group of options for a Tracer.
type TracerConfig struct {
	instrumentationVersion string
	// Schema URL of the telemetry emitted by the Tracer.
	schemaURL string
	attrs     attribute.Set
}

// InstrumentationVersion returns the version of the library providing instrumentation.
func (t *TracerConfig) InstrumentationVersion() string {
	return t.instrumentationVersion
}

// InstrumentationAttributes returns the attributes associated with the library
// providing instrumentation.
func (t *TracerConfig) InstrumentationAttributes() attribute.Set {
	return t.attrs
}

// SchemaURL returns the Schema URL of the telemetry emitted by the Tracer.
func (t *TracerConfig) SchemaURL() string {
	return t.schemaURL
}

// NewTracerConfig applies all the options to a returned TracerConfig.
func NewTracerConfig(options ...TracerOption) TracerConfig {
	var config TracerConfig
	for _, option := range options {
		config = option.apply(config)
	}
	return config
}
