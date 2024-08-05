# Telemetry System Documentation

## Overview

Our new telemetry system provides a unified interface for instrumenting applications with tracing, metrics, and logging. It supports multiple backends, including OpenTelemetry and Application Insights, allowing you to easily switch between them based on your needs.

## Getting Started

To use the telemetry system in your application, you first need to initialize it:

```go
import "your-project/telemetry"

func main() {
    t, err := telemetry.NewTelemetry()
    if err != nil {
        log.Fatalf("Failed to initialize telemetry: %v", err)
    }
    defer t.Shutdown(context.Background())

    // Your application code here
}
```

## Common Operations

### Starting and Ending Spans

Spans are used to trace the execution of your code:

```go
ctx := context.Background()
ctx, span := t.StartSpan(ctx, "operation-name")
defer t.EndSpan(span)

// Code for this operation
```

### Recording Metrics

You can record various types of metrics:

```go
// Record a simple metric
t.RecordMetric(ctx, "requests.count", 1)

// Increment a counter
t.IncrementCounter(ctx, "requests.total", 1)

// Record a gauge
t.RecordGauge(ctx, "queue.size", 42)
```

### Logging

The system provides methods for logging at different levels:

```go
t.LogInfo(ctx, "Operation completed successfully", attribute.Int("items_processed", 100))
t.LogWarning(ctx, "Resource usage high", attribute.Float64("cpu_usage", 0.95))
t.LogError(ctx, "Operation failed", err, attribute.String("operation", "data_export"))
```

### Posting Events

You can post custom events:

```go
t.PostEvent(ctx, "UserLoggedIn", map[string]string{
    "userId": "12345",
    "loginMethod": "oauth",
})
```

### Tracking Requests and Dependencies

For web applications and services, you can track HTTP requests and dependencies:

```go
start := time.Now()
// ... perform the request ...
duration := time.Since(start)
t.TrackRequest(ctx, "GET", "/api/data", duration, "200")

t.TrackDependency(ctx, "sql", "user_db", duration, true)
```

## Configuration

The telemetry system can be configured using environment variables:

- `TELEMETRY_TYPE`: Set to either "opentelemetry" or "appinsights" to choose the backend.
- `SERVICE_NAME`: The name of your service, used to identify the source of telemetry data.
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`: The endpoint for exporting OpenTelemetry traces.
- `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`: The endpoint for exporting OpenTelemetry metrics.
- `OTEL_TRACE_ENABLED`: Enables OpenTelemetry traces.
- `OTEL_METRICS_ENABLED`: Enables OpenTelemetry metrics.
- `APPINSIGHTS_INSTRUMENTATIONKEY`: The instrumentation key for Application Insights.

### Switching Backends

To switch between OpenTelemetry and Application Insights, simply change the `TELEMETRY_TYPE` environment variable:

```bash
# For OpenTelemetry
export TELEMETRY_TYPE=opentelemetry
export SERVICE_NAME=my-service
export OTEL_TRACE_ENABLED=true
export OTEL_METRICS_ENABLED=true
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://otel-collector:4317
export OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://otel-collector:4317

# For Application Insights
export TELEMETRY_TYPE=appinsights
export SERVICE_NAME=my-service
export APPINSIGHTS_INSTRUMENTATIONKEY=your-instrumentation-key
```

## Best Practices

1. Always use contexts: Pass the context through your application to maintain trace continuity.
2. Be consistent with metric names: Use a naming convention for your metrics to make them easier to query and analyze.
3. Add relevant attributes: When logging or recording metrics, include attributes that will help you debug issues or analyze performance.
4. Handle errors: Always check for and handle errors when initializing the telemetry system.
5. Use defer for EndSpan: This ensures spans are properly closed even if the function returns early due to an error.

## Extending the System

If you need to add custom functionality or support for a new backend, you can extend the `Telemetry` interface and create a new implementation. Then, update the `NewTelemetry()` function to return your new implementation based on the configuration.

## Troubleshooting

If you're not seeing telemetry data:
1. Check that the telemetry system is properly initialized.
2. Verify that your environment variables are correctly set.
3. Ensure that your telemetry backend (OpenTelemetry collector or Application Insights) is properly configured and accessible.
4. Check for any error messages in your application logs related to telemetry initialization or export.

For more detailed troubleshooting, you can enable debug logging in your application and inspect the telemetry-related log messages.

## Conclusion

This telemetry system provides a flexible and powerful way to instrument your Go applications. By using this unified interface, you can easily collect important metrics, traces, and logs, regardless of the backend system you choose to use. Always consider what data will be most valuable for monitoring and debugging your application, and make use of the various telemetry methods provided to capture that data.