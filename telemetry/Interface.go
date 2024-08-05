package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	// Shutdown shuts down the telemetry provider
	Shutdown(ctx context.Context)
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
// based on the configuration specified in environment variables.
func NewTelemetry() (Telemetry, error) {
	telemetryType := getTelemetryType()

	switch telemetryType {
	case TelemetryTypeOpenTelemetry:
		traceEnabled := os.Getenv("OTEL_TRACE_ENABLED") == "true"
		metricsEnabled := os.Getenv("OTEL_METRICS_ENABLED") == "true"
		serviceName := os.Getenv("SERVICE_NAME")
		if serviceName == "" {
			serviceName = "unknown-service"
		}
		traceEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
		metricEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
		return newOpenTelemetry(serviceName, traceEndpoint, metricEndpoint, traceEnabled, metricsEnabled)

	case TelemetryTypeAppInsights:
		return newAppInsightsTelemetry()

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
