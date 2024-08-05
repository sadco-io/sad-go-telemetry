package telemetry

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
	"github.com/sadco-io/sad-go-logger/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	APPINSIGHTS_INSTRUMENTATIONKEY_DEFAULT = "6b57af63-7a39-4834-9d3d-405ddb07a51a"
)

type AppInsightsTelemetry struct {
	client           appinsights.TelemetryClient
	hostname         string
	serviceName      string
	traceEnabled     bool
	telemetryEnabled bool
}

func newAppInsightsTelemetry() (Telemetry, error) {
	iKey := os.Getenv("APPINSIGHTS_INSTRUMENTATIONKEY")
	if iKey == "" {
		iKey = APPINSIGHTS_INSTRUMENTATIONKEY_DEFAULT
		logger.Log.Info("APPINSIGHTS_INSTRUMENTATIONKEY is not set, using default instrumentation key")
	}

	client := appinsights.NewTelemetryClient(iKey)

	hostname, err := os.Hostname()
	if err != nil {
		logger.Log.Warn("Error retrieving hostname", zap.Error(err))
		hostname = "unkw"
	}

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		logger.Log.Info("SERVICE_NAME is not set, using Melina as default")
		serviceName = "Melina"
	}

	traceEnabled, _ := strconv.ParseBool(os.Getenv("APPINSIGHTS_TRACE_ENABLED"))
	telemetryEnabled, _ := strconv.ParseBool(os.Getenv("APPINSIGHTS_ENABLED"))

	return &AppInsightsTelemetry{
		client:           client,
		hostname:         hostname,
		serviceName:      serviceName,
		traceEnabled:     traceEnabled,
		telemetryEnabled: telemetryEnabled,
	}, nil
}

func (ai *AppInsightsTelemetry) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	// App Insights doesn't have a direct equivalent to OpenTelemetry's spans
	// We'll create a custom span-like structure
	span := &appInsightsSpan{
		name:      name,
		startTime: time.Now(),
	}
	return context.WithValue(ctx, "appinsights_span", span), span
}

func (ai *AppInsightsTelemetry) EndSpan(span trace.Span) {
	if aiSpan, ok := span.(*appInsightsSpan); ok {
		duration := time.Since(aiSpan.startTime)
		ai.PostEvent(aiSpan.name, map[string]string{"duration_ms": strconv.FormatInt(duration.Milliseconds(), 10)})
	}
}

func (ai *AppInsightsTelemetry) AddEvent(span trace.Span, name string, attributes ...attribute.KeyValue) {
	properties := make(map[string]string)
	for _, attr := range attributes {
		properties[string(attr.Key)] = attr.Value.Emit()
	}
	ai.PostEvent(name, properties)
}

func (ai *AppInsightsTelemetry) RecordMetric(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue) {
	ai.PostMetrics(name, value)
	for _, attr := range attributes {
		ai.PostMetrics(name+"."+string(attr.Key), attr.Value.AsFloat64())
	}
}

func (ai *AppInsightsTelemetry) IncrementCounter(ctx context.Context, name string, increment float64, attributes ...attribute.KeyValue) {
	ai.RecordMetric(ctx, name, increment, attributes...)
}

func (ai *AppInsightsTelemetry) RecordGauge(ctx context.Context, name string, value float64, attributes ...attribute.KeyValue) {
	ai.RecordMetric(ctx, name, value, attributes...)
}

func (ai *AppInsightsTelemetry) LogInfo(ctx context.Context, message string, attributes ...attribute.KeyValue) {
	properties := make(map[string]string)
	for _, attr := range attributes {
		properties[string(attr.Key)] = attr.Value.Emit()
	}
	ai.PostTrace(message, contracts.Information, properties)
}

func (ai *AppInsightsTelemetry) LogWarning(ctx context.Context, message string, attributes ...attribute.KeyValue) {
	properties := make(map[string]string)
	for _, attr := range attributes {
		properties[string(attr.Key)] = attr.Value.Emit()
	}
	ai.PostTrace(message, contracts.Warning, properties)
}

func (ai *AppInsightsTelemetry) LogError(ctx context.Context, message string, err error, attributes ...attribute.KeyValue) {
	properties := make(map[string]string)
	for _, attr := range attributes {
		properties[string(attr.Key)] = attr.Value.Emit()
	}
	properties["error"] = err.Error()
	ai.PostTrace(message, contracts.Error, properties)
}

func (ai *AppInsightsTelemetry) TrackRequest(ctx context.Context, method, url string, duration time.Duration, responseCode string) {
	if ai.telemetryEnabled {
		request := appinsights.NewRequestTelemetry(method, url, duration, responseCode)
		ai.client.Track(request)
	}
}

func (ai *AppInsightsTelemetry) TrackDependency(ctx context.Context, dependencyType, target string, duration time.Duration, success bool) {
	if ai.telemetryEnabled {
		dependency := appinsights.NewRemoteDependencyTelemetry(dependencyType, target, "", duration, success)
		ai.client.Track(dependency)
	}
}

func (ai *AppInsightsTelemetry) TrackAvailability(ctx context.Context, name string, duration time.Duration, success bool) {
	if ai.telemetryEnabled {
		availability := appinsights.NewAvailabilityTelemetry(name, duration, success)
		ai.client.Track(availability)
	}
}

func (ai *AppInsightsTelemetry) SetUser(ctx context.Context, id string) {
	ai.client.Context().User().SetId(id)
}

func (ai *AppInsightsTelemetry) SetSession(ctx context.Context, id string) {
	ai.client.Context().Session().SetId(id)
}

func (ai *AppInsightsTelemetry) Shutdown(ctx context.Context) {
	select {
	case <-ai.client.Channel().Close(10 * time.Second):
		logger.Log.Info("AppInsights telemetry channel has been successfully closed")
	case <-ctx.Done():
		logger.Log.Error("Failed to close AppInsights telemetry channel before context deadline")
	}
}

func (ai *AppInsightsTelemetry) PostMetrics(name string, value float64) {
	if ai.telemetryEnabled {
		go func() {
			metricTelemetry := appinsights.NewMetricTelemetry(name, value)
			ai.client.Track(metricTelemetry)
		}()
	}
}

func (ai *AppInsightsTelemetry) PostEvent(name string, properties map[string]string) {
	if ai.telemetryEnabled {
		go func() {
			eventTelemetry := appinsights.NewEventTelemetry(name)
			for k, v := range properties {
				eventTelemetry.Properties[k] = v
			}
			ai.client.Track(eventTelemetry)
		}()
	}
}

func (ai *AppInsightsTelemetry) PostTrace(message string, severity contracts.SeverityLevel, properties map[string]string) {
	if ai.traceEnabled && ai.telemetryEnabled {
		go func() {
			newMessage := ai.serviceName + "_" + ai.hostname + "_" + message
			trace := appinsights.NewTraceTelemetry(newMessage, severity)
			for key, value := range properties {
				trace.Properties[key] = value
			}
			ai.client.Track(trace)
		}()
	}
}

// Custom span-like structure for App Insights
type appInsightsSpan struct {
	name      string
	startTime time.Time
}
