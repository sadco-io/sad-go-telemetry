package telemetry

import "github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"

// TelemetryAppInsights defines the interface for combined telemetry and application insights functionality
type TelemetryAppInsights interface {
	NewTelemetryAppInsights(appInsightsKey string, hostName string, serviceName string, traceEnabled bool, telemetryEnabled bool)
	GetTelemetryAppInsights()
	PostMetrics(name string, value float64)
	PostEvent(name string, properties map[string]string)
	PostTrace(message string, severity contracts.SeverityLevel, properties map[string]string)
	Close()
}
