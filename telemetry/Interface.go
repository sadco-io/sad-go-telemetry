// telemetry/Interface.go

package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Telemetry defines the interface for telemetry operations
type Telemetry interface {
	// StartSpan starts a new span and returns the context and the span
	StartSpan(ctx context.Context, name string) (context.Context, trace.Span)

	// EndSpan ends the given span
	EndSpan(span trace.Span)

	// AddEvent adds an event to the given span
	AddEvent(span trace.Span, name string, attributes ...attribute.KeyValue)

	// RecordMetric records a metric with the given name and value
	RecordMetric(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue)

	// PostEvent posts an event with the given name and properties
	PostEvent(name string, properties map[string]string)

	// PostTrace posts a trace message with the given severity and properties
	PostTrace(message string, severity string, properties map[string]string)

	// RecordError records an error with the given attributes
	RecordError(ctx context.Context, err error, attributes ...attribute.KeyValue)

	// IncrementCounter increments a counter metric
	IncrementCounter(ctx context.Context, name string, increment float64, attributes ...attribute.KeyValue)

	// RecordGauge records a gauge metric
	RecordGauge(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue)

	// LogInfo logs an info message
	LogInfo(ctx context.Context, message string, attributes ...attribute.KeyValue)

	// LogWarning logs a warning message
	LogWarning(ctx context.Context, message string, attributes ...attribute.KeyValue)

	// LogError logs an error message
	LogError(ctx context.Context, message string, err error, attributes ...attribute.KeyValue)

	// TrackRequest records an HTTP request as a span
	TrackRequest(ctx context.Context, method, url string, duration time.Duration, statusCode int)

	// TrackDependency records a dependency call as a span
	TrackDependency(ctx context.Context, dependencyType, target string, duration time.Duration, success bool)

	// TrackAvailability records an availability test
	TrackAvailability(ctx context.Context, name string, duration time.Duration, success bool)

	// SetUser sets the user ID for the current context
	SetUser(ctx context.Context, id string)

	// SetSession sets the session ID for the current context
	SetSession(ctx context.Context, id string)

	// Shutdown shuts down the telemetry provider
	Shutdown(ctx context.Context) error
}

// TelemetryType represents the type of telemetry implementation to use
type TelemetryType string

const (
	// TelemetryTypeOpenTelemetry represents the OpenTelemetry implementation
	TelemetryTypeOpenTelemetry TelemetryType = "opentelemetry"
	// TelemetryTypeAppInsights represents the Application Insights implementation
	TelemetryTypeAppInsights TelemetryType = "appinsights"
)

// NewTelemetry creates and returns the appropriate telemetry implementation
func NewTelemetry() (Telemetry, error) {
	telemetryType := os.Getenv("TELEMETRY_TYPE")
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "unknown-service"
	}

	switch telemetryType {
	case "opentelemetry", "otel", "":
		traceEnabled := os.Getenv("OTEL_TRACE_ENABLED") == "true"
		metricsEnabled := os.Getenv("OTEL_METRICS_ENABLED") == "true"
		traceEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
		metricEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
		return NewOpenTelemetry(serviceName, traceEndpoint, metricEndpoint, traceEnabled, metricsEnabled)
	// Add cases for other telemetry types if needed
	default:
		return nil, fmt.Errorf("unknown telemetry type: %s", telemetryType)
	}
}

// getTelemetryType determines the telemetry type to use based on environment variables
func getTelemetryType() TelemetryType {
	telemetryType := os.Getenv("TELEMETRY_TYPE")
	if telemetryType == "" {
		// Default to OpenTelemetry if not specified
		return TelemetryTypeOpenTelemetry
	}

	switch strings.ToLower(telemetryType) {
	case "opentelemetry", "otel":
		return TelemetryTypeOpenTelemetry
	case "appinsights", "applicationinsights":
		return TelemetryTypeAppInsights
	default:
		// If an unknown type is specified, default to OpenTelemetry
		return TelemetryTypeOpenTelemetry
	}
}
