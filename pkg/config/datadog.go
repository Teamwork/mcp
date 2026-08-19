package config

import (
	"log/slog"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
)

func startDatadog(resources Resources) error {
	return tracer.Start(
		tracer.WithAgentAddr(resources.Info.DatadogAPM.AgentHost+":"+resources.Info.DatadogAPM.AgentPort),
		tracer.WithDogstatsdAddr(resources.Info.DatadogAPM.AgentHost+":"+resources.Info.DatadogAPM.StatsdPort),
		tracer.WithEnv(resources.Info.DatadogAPM.Environment),
		tracer.WithService(resources.Info.DatadogAPM.Service),
		tracer.WithServiceVersion(resources.Info.DatadogAPM.Version),
		tracer.WithLogger(newDatadogLogger(resources.logger)),
		tracer.WithGlobalTag("awsregion", resources.Info.AWSRegion),
		tracer.WithRuntimeMetrics(),
	)
}

type datadogLogger struct {
	logger *slog.Logger
}

func newDatadogLogger(logger *slog.Logger) datadogLogger {
	return datadogLogger{
		logger: logger,
	}
}

func (d datadogLogger) Log(msg string) {
	if d.logger == nil {
		return
	}
	d.logger.Info(msg)
}
