# SAD Go Telemetry

SAD Go Telemetry is a comprehensive, OpenTelemetry-based instrumentation package for Go applications. It provides easy-to-use APIs for tracing and metrics, supporting various backends through the OpenTelemetry protocol.

## Features

* Built on the OpenTelemetry standard
* Support for both tracing and metrics
* Configurable through environment variables
* Easy-to-use API for starting spans and recording metrics
* Automatic context propagation
* Support for custom attributes and events in spans
* Graceful shutdown of telemetry providers

## Installation

```
go get github.com/sadco-io/sad-go-telemetry
```

## Usage

Import the package in your Go code:

```go
import "github.com/sadco-io/sad-go-telemetry/telemetry"
```

Start a new span:

```go
ctx, span := telemetry.StartSpan(ctx, "operation_name")
defer telemetry.EndSpan(span)
```

Add an event to a span:

```go
telemetry.AddEvent(span, "interesting_event", attribute.String("key", "value"))
```

Record a metric:

```go
telemetry.RecordMetric(ctx, "metric_name", 1.0, attribute.String("key", "value"))
```

Shutdown telemetry providers:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
telemetry.Shutdown(ctx)
```

## Configuration

The telemetry module is configured using environment variables. Here's a list of available options:

### General Configuration

* `SERVICE_NAME`: Name of your service (default: "unknown-service")
* `OTEL_TRACE_ENABLED`: Set to "true" to enable tracing (default: false)
* `OTEL_METRICS_ENABLED`: Set to "true" to enable metrics (default: false)

### Exporter Configuration

* `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`: Endpoint for the trace exporter
* `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`: Endpoint for the metrics exporter

## Example Configuration

Here's an example of how to configure the telemetry module:

```bash
export SERVICE_NAME="my-awesome-service"
export OTEL_TRACE_ENABLED="true"
export OTEL_METRICS_ENABLED="true"
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="http://localhost:4317"
export OTEL_EXPORTER_OTLP_METRICS_ENDPOINT="http://localhost:4317"
```

## Performance Considerations

* The telemetry module uses the OpenTelemetry SDK, which is designed for high performance.
* Tracing and metrics are collected in-memory and exported asynchronously to minimize performance impact.
* If an exporter is unavailable, telemetry data is buffered in memory, and the exporter will attempt to reconnect periodically.

## Thread Safety

The telemetry module is designed to be thread-safe and can be safely used from multiple goroutines concurrently.

## Extending the Telemetry Module

The telemetry module uses the OpenTelemetry SDK, which supports various exporters. You can extend the module by implementing new exporters or by configuring existing ones to send data to different backends.

## Contributing

Contributions to SAD Go Telemetry are welcome! Please submit pull requests with any enhancements, bug fixes, or new features.

## Support

For issues, feature requests, or questions, please file an issue on the GitHub repository.

## License

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

This project is licensed under the MIT License - see the [LICENSE] file for details.

Copyright (c) 2024 SAD co.


