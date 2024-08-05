// Package telemetry provides functionality for instrumenting applications
// with OpenTelemetry-based tracing and metrics. It offers simple APIs for
// starting spans, recording metrics, and managing the telemetry lifecycle.
package telemetry

// init initializes the telemetry package. It sets up the logger, reads
// configuration from environment variables, and initializes the OpenTelemetry/AppInsights
// provider.
func init() {

	NewTelemetry()

}
