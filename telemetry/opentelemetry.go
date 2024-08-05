package telemetry

import (
	"context"
	"time"

	"github.com/sadco-io/sad-go-logger/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Implementation of the Telemetry interface
type OpenTelemetry struct {
	traceEnabled   bool
	metricsEnabled bool
	tracer         trace.Tracer
	meter          metric.Meter
	traceProvider  *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

func newOpenTelemetry(serviceName, traceEndpoint, metricEndpoint string, traceEnabled bool, metricsEnabled bool) (Telemetry, error) {

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	var tp *sdktrace.TracerProvider
	var mp *sdkmetric.MeterProvider

	if traceEnabled {
		traceExporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(traceEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}

		tp := sdktrace.NewTracerProvider(
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
			return nil, err
		}

		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
	}

	tracer := otel.Tracer(serviceName)
	meter := otel.Meter(serviceName)

	return &OpenTelemetry{
		traceEnabled:   traceEnabled,
		metricsEnabled: metricsEnabled,
		tracer:         tracer,
		meter:          meter,
		traceProvider:  tp,
		meterProvider:  mp,
	}, nil
}
func (o *OpenTelemetry) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !traceEnabled {
		logger.Log.Info("Failed to start span, trace not enabled for this service")
		return ctx, nil
	}
	return o.tracer.Start(ctx, name)
}

func (o *OpenTelemetry) EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

func (o *OpenTelemetry) AddEvent(span trace.Span, name string, attributes ...attribute.KeyValue) {
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

func (o *OpenTelemetry) PostEvent(name string, properties map[string]string) {
	// Convert properties to OpenTelemetry attributes
	attrs := make([]attribute.KeyValue, 0, len(properties))
	for k, v := range properties {
		attrs = append(attrs, attribute.String(k, v))
	}

}

func (o *OpenTelemetry) RecordMetric(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue) {
	if !metricsEnabled {
		logger.Log.Info("Failed to create metric instrument, metrics not enabled for this service")
		return
	}
	instrument, err := o.meter.Float64Counter(name)
	if err != nil {
		logger.Log.Error("Failed to create metric instrument", zap.Error(err))
		return
	}
	instrument.Add(ctx, value, metric.WithAttributes(attributes...))
}

func (o *OpenTelemetry) IncrementCounter(ctx context.Context, name string, increment float64, attributes ...attribute.KeyValue) {
	o.RecordMetric(ctx, name, increment, attributes...)
}

func (o *OpenTelemetry) RecordGauge(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue) {
	instrument, err := o.meter.Float64ObservableGauge(name)
	if err != nil {
		logger.Log.Error("Failed to create gauge instrument", zap.Error(err))
		return
	}
	instrument.Observe(ctx, value, metric.WithAttributes(attributes...))
}

func (o *OpenTelemetry) LogInfo(ctx context.Context, message string, attributes ...attribute.KeyValue) {
	logger.Log.Info(message, zap.Any("attributes", attributes))
}

func (o *OpenTelemetry) LogWarning(ctx context.Context, message string, attributes ...attribute.KeyValue) {
	logger.Log.Warn(message, zap.Any("attributes", attributes))
}

func (o *OpenTelemetry) LogError(ctx context.Context, message string, err error, attributes ...attribute.KeyValue) {
	logger.Log.Error(message, zap.Error(err), zap.Any("attributes", attributes))
}

func (o *OpenTelemetry) TrackRequest(ctx context.Context, method, url string, duration time.Duration, responseCode string) {
	attributes := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.url", url),
		attribute.Int64("http.duration_ms", duration.Milliseconds()),
		attribute.String("http.status_code", responseCode),
	}
	o.RecordMetric(ctx, "http.requests", 1, attributes...)
}

func (o *OpenTelemetry) TrackDependency(ctx context.Context, dependencyType, target string, duration time.Duration, success bool) {
	attributes := []attribute.KeyValue{
		attribute.String("dependency.type", dependencyType),
		attribute.String("dependency.target", target),
		attribute.Int64("dependency.duration_ms", duration.Milliseconds()),
		attribute.Bool("dependency.success", success),
	}
	o.RecordMetric(ctx, "dependency.calls", 1, attributes...)
}

func (o *OpenTelemetry) TrackAvailability(ctx context.Context, name string, duration time.Duration, success bool) {
	attributes := []attribute.KeyValue{
		attribute.String("availability.test", name),
		attribute.Int64("availability.duration_ms", duration.Milliseconds()),
		attribute.Bool("availability.success", success),
	}
	o.RecordMetric(ctx, "availability.tests", 1, attributes...)
}

func (o *OpenTelemetry) SetUser(ctx context.Context, id string) {
	// Add user information as an attribute to the current span
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attribute.String("user.id", id))
	}
}

func (o *OpenTelemetry) SetSession(ctx context.Context, id string) {
	// Add session information as an attribute to the current span
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attribute.String("session.id", id))
	}
}

type SeverityLevel int

const (
	Verbose SeverityLevel = iota
	Information
	Warning
	Error
	Critical
)

func (o *OpenTelemetry) PostTrace(message string, severity string, properties map[string]string) {
	// Convert properties to OpenTelemetry attributes
	attrs := make([]attribute.KeyValue, 0, len(properties)+1)
	attrs = append(attrs, attribute.String("message", message))
	for k, v := range properties {
		attrs = append(attrs, attribute.String(k, v))
	}
	logger.Log.Info("Event recorded", zap.Any("attributes", attrs))
}

func (o *OpenTelemetry) Shutdown(ctx context.Context) {
	if err := o.traceProvider.Shutdown(ctx); err != nil {
		logger.Log.Error("Error shutting down trace provider", zap.Error(err))
	}
	if err := o.meterProvider.Shutdown(ctx); err != nil {
		logger.Log.Error("Error shutting down meter provider", zap.Error(err))
	}
}
