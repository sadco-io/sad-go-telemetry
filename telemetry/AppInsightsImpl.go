package telemetry

import (
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
	"github.com/sadco-io/sad-go-logger/logger"
)

type TelemetryAppInsights struct {
	appInsights      appinsights.TelemetryClient
	serviceName      string
	hostName         string
	traceEnabled     bool
	telemetryEnabled bool
	client           appinsights.TelemetryClient
}

func NewTelemetryAppInsights(appInsightsKey string, hostName string, serviceName string, traceEnabled bool, telemetryEnabled bool) (*TelemetryAppInsights, error) {

	tai := &TelemetryAppInsights{}
	if appInsightsKey != "" {
		tai.appInsights = appinsights.NewTelemetryClient(appInsightsKey)
	} else {
		logger.Log.Warn("APPINSIGHTS_INSTRUMENTATIONKEY is not set, Application Insights functionality will be limited")
	}
	client := appinsights.NewTelemetryClient(appInsightsKey)
	tai.client = client
	tai.hostName = hostName
	tai.serviceName = serviceName
	tai.traceEnabled = traceEnabled
	tai.telemetryEnabled = telemetryEnabled

	return tai, nil
}

func GetTelemetryAppInsights() *TelemetryAppInsights {
	return &TelemetryAppInsights{}
}

func (tai *TelemetryAppInsights) PostMetrics(name string, value float64) {
	if tai.telemetryEnabled {
		go func() {
			metricTelemetry := appinsights.NewMetricTelemetry(name, value)
			tai.client.Track(metricTelemetry)
		}()
	}
}

func (tai *TelemetryAppInsights) PostEvent(name string, properties map[string]string) {
	if tai.telemetryEnabled {
		go func() {
			eventTelemetry := appinsights.NewEventTelemetry(name)
			for k, v := range properties {
				eventTelemetry.Properties[k] = v
			}
			tai.client.Track(eventTelemetry)
		}()
	}
}

func (tai *TelemetryAppInsights) PostTrace(message string, severity contracts.SeverityLevel, properties map[string]string) {
	if tai.traceEnabled && tai.telemetryEnabled {
		go func() {
			newMessage := tai.serviceName + "_" + tai.hostName + "_" + message
			trace := appinsights.NewTraceTelemetry(newMessage, severity)
			for key, value := range properties {
				trace.Properties[key] = value
			}
			tai.client.Track(trace)
		}()
	}
}

func (tai *TelemetryAppInsights) Close() {
	select {
	case <-tai.client.Channel().Close(10):
	case <-time.After(10 * time.Second):
		logger.Log.Error("AppInsights timeout error.")
	}
	logger.Log.Sync() // flush the logger's outstanding messages before exit
}
