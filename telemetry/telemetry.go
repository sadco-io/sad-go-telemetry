// Package telemetry provides functionality for instrumenting applications
// with OpenTelemetry-based tracing and metrics. It offers simple APIs for
// starting spans, recording metrics, and managing the telemetry lifecycle.
package telemetry

import (
	"context"
	"os"

	"github.com/sadco-io/sad-go-logger/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	// tracer is the global tracer used for creating spans.
	tracer trace.Tracer

	// meter is the global meter used for recording metrics.
	meter metric.Meter

	// serviceName is the name of the current service, used for identifying
	// the source of telemetry data.
	serviceName string

	// traceEnabled indicates whether tracing is enabled for this service.
	traceEnabled bool

	// metricsEnabled indicates whether metrics collection is enabled for this service.
	metricsEnabled bool

	// traceEndpoint is the endpoint URL for the trace exporter.
	traceEndpoint string

	// metricEndpoint is the endpoint URL for the metric exporter.
	metricEndpoint string
)

// init initializes the telemetry package. It sets up the logger, reads
// configuration from environment variables, and initializes the OpenTelemetry
// provider.
func init() {

	// Set the service name from environment variable
	serviceName = os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "unknown-service"
		logger.Warn("SERVICE_NAME is not set, using 'unknown-service' as default")
	}

	// Configure tracing and metrics based on environment variables
	traceEnabled = os.Getenv("OTEL_TRACE_ENABLED") == "true"
	metricsEnabled = os.Getenv("OTEL_METRICS_ENABLED") == "true"
	traceEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	metricEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
	instrument := os.Getenv("INSTRUMENT")

	// Initialize the OpenTelemetry provider
	if err := initProvider(); err != nil {
		logger.Error("Failed to initialize OpenTelemetry provider", zap.Error(err))
	}

	// Create global tracer and meter instances
	tracer = otel.Tracer(serviceName)
	meter = otel.Meter(serviceName)

	// Initialize AppInsights if enabled
	if instrument == "APPINSIGHTS" {
		iKey := os.Getenv("APPINSIGHTS_INSTRUMENTATIONKEY")
		hostName, err := os.Hostname()
		if err != nil {
			logger.Log.Warn("Error retrieving hostname: %v", zap.Error(err))
			logger.Log.Warn("Setting hostname to unkw")
			hostName = "unkw"
		}
		NewTelemetryAppInsights(iKey, hostName, serviceName, traceEnabled, metricsEnabled)
	}
}

// initProvider sets up the OpenTelemetry trace and metric providers based on
// the configuration. It creates exporters, sets up providers, and registers
// them with the global OpenTelemetry API.
func initProvider() error {
	ctx := context.Background()

	// Create a new resource with the service name
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return err
	}

	// Initialize trace provider if tracing is enabled
	if traceEnabled {
		var traceExporter *otlptrace.Exporter
		var err error

		// Create trace exporter with custom endpoint if specified
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithInsecure(),
		}
		if traceEndpoint != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(traceEndpoint))
		}

		client := otlptracegrpc.NewClient(opts...)
		traceExporter, err = otlptrace.New(ctx, client)

		if err != nil {
			return err
		}

		// Create and set the trace provider
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
	}

	// Initialize metric provider if metrics are enabled
	if metricsEnabled {
		var metricExporter *otlpmetricgrpc.Exporter
		var err error

		// Create metric exporter with custom endpoint if specified
		if metricEndpoint != "" {
			metricExporter, err = otlpmetricgrpc.New(ctx,
				otlpmetricgrpc.WithEndpoint(metricEndpoint),
				otlpmetricgrpc.WithInsecure(),
			)
		} else {
			metricExporter, err = otlpmetricgrpc.New(ctx)
		}

		if err != nil {
			return err
		}

		// Create and set the meter provider
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
	}

	return nil
}

// StartSpan starts a new span and returns the context and the span.
// If tracing is disabled, it returns the original context and a nil span.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !traceEnabled {
		return ctx, nil
	}
	return tracer.Start(ctx, name)
}

// EndSpan ends the given span
func EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

// AddEvent adds an event to the given span
func AddEvent(span trace.Span, name string, attributes ...attribute.KeyValue) {
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

// RecordMetric records a metric with the given name and value.
// Additional attributes can be provided to add more context to the metric.
func RecordMetric(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue) {
	if !metricsEnabled {
		return
	}

	instrument, err := meter.Float64Counter(name)
	if err != nil {
		logger.Error("Failed to create metric instrument", zap.Error(err))
		return
	}

	instrument.Add(ctx, value, metric.WithAttributes(attributes...))
}

// Shutdown shuts down the OpenTelemetry provider
func Shutdown(ctx context.Context) {
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}

	if mp, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); ok {
		if err := mp.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down meter provider", zap.Error(err))
		}
	}
}
