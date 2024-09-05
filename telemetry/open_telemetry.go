// open_telemetry.go - Implementation of the Telemetry interface using OpenTelemetry

package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/sadco-io/sad-go-logger/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// OpenTelemetry implements the Telemetry interface
type OpenTelemetry struct {
	tracer         trace.Tracer
	meter          metric.Meter
	traceProvider  *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	traceEnabled   bool
	metricsEnabled bool
}

// NewOpenTelemetry creates and initializes a new OpenTelemetry instance
func NewOpenTelemetry(serviceName, traceEndpoint, metricEndpoint string, traceEnabled, metricsEnabled bool) (*OpenTelemetry, error) {
	logger.Log.Info("OpenTelemetry Configuration ",
		zap.String("serviceName", serviceName),
		zap.String("traceEndpoint", traceEndpoint),
		zap.String("metricEndpoint", metricEndpoint),
		zap.Bool("traceEnabled", traceEnabled),
		zap.Bool("metricsEnabled", metricsEnabled))
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var tp *sdktrace.TracerProvider
	var mp *sdkmetric.MeterProvider

	if traceEnabled {
		traceExporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(traceEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			logger.Log.Error("Failed to create OpenTelemetry exporter",
				zap.Error(err),
				zap.String("endpoint", metricEndpoint),
				zap.Bool("metricsEnabled", metricsEnabled))
			return nil, fmt.Errorf("failed to create metric exporter: %w", err)
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
	}

	if metricsEnabled {
		metricExporter, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(metricEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create metric exporter: %w", err)
		}

		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
	}

	tracer := otel.Tracer(serviceName)
	meter := otel.Meter(serviceName)

	return &OpenTelemetry{
		tracer:         tracer,
		meter:          meter,
		traceProvider:  tp,
		meterProvider:  mp,
		traceEnabled:   traceEnabled,
		metricsEnabled: metricsEnabled,
	}, nil
}

// StartSpan starts a new span and returns the context and the span
func (o *OpenTelemetry) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !o.traceEnabled {
		return ctx, nil
	}
	return o.tracer.Start(ctx, name)
}

// EndSpan ends the given span
func (o *OpenTelemetry) EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

// AddEvent adds an event to the given span
func (o *OpenTelemetry) AddEvent(span trace.Span, name string, attributes ...attribute.KeyValue) {
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

// PostEvent posts an event with the given name and properties
func (o *OpenTelemetry) PostEvent(name string, properties map[string]string) {
	if !o.traceEnabled {
		return
	}
	attrs := make([]attribute.KeyValue, 0, len(properties))
	for k, v := range properties {
		attrs = append(attrs, attribute.String(k, v))
	}
	span := trace.SpanFromContext(context.Background())
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RecordMetric records a metric with the given name and value
func (o *OpenTelemetry) RecordMetric(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue) {
	if !o.metricsEnabled {
		return
	}
	instrument, err := o.meter.Float64Counter(name)
	if err != nil {
		logger.Log.Error("Failed to create metric instrument", zap.Error(err))
		return
	}
	instrument.Add(ctx, value, metric.WithAttributes(attributes...))
}

// RecordError records an error as a span event and sets the span status
func (o *OpenTelemetry) RecordError(ctx context.Context, err error, attributes ...attribute.KeyValue) {
	if !o.traceEnabled || err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err, trace.WithAttributes(attributes...))
		span.SetStatus(codes.Error, err.Error())
	}
}

// IncrementCounter increments a counter metric
func (o *OpenTelemetry) IncrementCounter(ctx context.Context, name string, increment float64, attributes ...attribute.KeyValue) {
	o.RecordMetric(ctx, name, increment, attributes...)
}

func (o *OpenTelemetry) RecordGauge(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue) {
	if !o.metricsEnabled {
		return
	}

	instrument, err := o.meter.Float64Gauge(name)
	if err != nil {
		logger.Log.Error("Failed to create gauge instrument", zap.Error(err))
		return
	}

	instrument.Record(ctx, value, metric.WithAttributes(attributes...))
}

// LogInfo logs an info message
func (o *OpenTelemetry) LogInfo(ctx context.Context, message string, attributes ...attribute.KeyValue) {
	logger.Log.Info(message, zap.Any("attributes", attributes))
}

// LogWarning logs a warning message
func (o *OpenTelemetry) LogWarning(ctx context.Context, message string, attributes ...attribute.KeyValue) {
	logger.Log.Warn(message, zap.Any("attributes", attributes))
}

// LogError logs an error message
func (o *OpenTelemetry) LogError(ctx context.Context, message string, err error, attributes ...attribute.KeyValue) {
	logger.Log.Error(message, zap.Error(err), zap.Any("attributes", attributes))
	o.RecordError(ctx, err, attributes...)
}

// TrackRequest records an HTTP request as a span
func (o *OpenTelemetry) TrackRequest(ctx context.Context, method, url string, duration time.Duration, statusCode int) {
	if !o.traceEnabled {
		return
	}
	ctx, span := o.StartSpan(ctx, "HTTP Request")
	defer o.EndSpan(span)

	span.SetAttributes(
		semconv.HTTPMethodKey.String(method),
		semconv.HTTPURLKey.String(url),
		semconv.HTTPStatusCodeKey.Int(statusCode),
	)
	span.SetStatus(httpStatusToSpanStatus(statusCode))
	span.SetAttributes(attribute.Int64("http.duration_ms", duration.Milliseconds()))
}

// TrackDependency records a dependency call as a span
func (o *OpenTelemetry) TrackDependency(ctx context.Context, dependencyType, target string, duration time.Duration, success bool) {
	if !o.traceEnabled {
		return
	}
	ctx, span := o.StartSpan(ctx, "Dependency Call")
	defer o.EndSpan(span)

	span.SetAttributes(
		attribute.String("dependency.type", dependencyType),
		attribute.String("dependency.target", target),
		attribute.Int64("dependency.duration_ms", duration.Milliseconds()),
		attribute.Bool("dependency.success", success),
	)
	if !success {
		span.SetStatus(codes.Error, "Dependency call failed")
	}
}

// TrackAvailability records an availability test
func (o *OpenTelemetry) TrackAvailability(ctx context.Context, name string, duration time.Duration, success bool) {
	if !o.metricsEnabled {
		return
	}
	attributes := []attribute.KeyValue{
		attribute.String("availability.test", name),
		attribute.Int64("availability.duration_ms", duration.Milliseconds()),
		attribute.Bool("availability.success", success),
	}
	o.RecordMetric(ctx, "availability.tests", 1, attributes...)
}

// SetUser sets the user ID for the current context
func (o *OpenTelemetry) SetUser(ctx context.Context, id string) {
	if !o.traceEnabled {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attribute.String("user.id", id))
	}
}

// SetSession sets the session ID for the current context
func (o *OpenTelemetry) SetSession(ctx context.Context, id string) {
	if !o.traceEnabled {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attribute.String("session.id", id))
	}
}

// PostTrace posts a trace message with the given severity and properties
func (o *OpenTelemetry) PostTrace(message string, severity string, properties map[string]string) {
	if !o.traceEnabled {
		return
	}
	attrs := make([]attribute.KeyValue, 0, len(properties)+2)
	attrs = append(attrs, attribute.String("message", message), attribute.String("severity", severity))
	for k, v := range properties {
		attrs = append(attrs, attribute.String(k, v))
	}
	span := trace.SpanFromContext(context.Background())
	if span.IsRecording() {
		span.AddEvent("Trace", trace.WithAttributes(attrs...))
	}
	logger.Log.Info("Trace recorded", zap.Any("attributes", attrs))
}

// Shutdown shuts down the telemetry provider
func (o *OpenTelemetry) Shutdown(ctx context.Context) error {
	var err error
	if o.traceProvider != nil {
		err = o.traceProvider.Shutdown(ctx)
	}
	if o.meterProvider != nil {
		if mErr := o.meterProvider.Shutdown(ctx); mErr != nil {
			err = fmt.Errorf("trace: %v, metric: %w", err, mErr)
		}
	}
	return err
}

// httpStatusToSpanStatus converts an HTTP status code to a span status
func httpStatusToSpanStatus(statusCode int) (codes.Code, string) {
	if statusCode >= 400 {
		return codes.Error, fmt.Sprintf("HTTP %d", statusCode)
	}
	return codes.Ok, ""
}
