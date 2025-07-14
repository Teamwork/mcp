package config

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// Resources stores all the resources loaded in the startup.
type Resources struct {
	teamworkHTTPClient *http.Client
	teamworkEngine     *twapi.Engine
	logger             *slog.Logger

	// Info stores environment variables mappings.
	Info struct {
		// ServerAddress is the address of the server.
		ServerAddress string
		// Environment is the environment this app is running in.
		Environment string
		// DevEnvInstallation is the Teamwork DevEnv installation URL.
		DevEnvInstallation string
		// DatadogAPMService is the Datadog APM service name.
		DatadogAPMService string
		// HAProxyURL is the URL of the HAProxy instance.
		HAProxyURL string
	}
}

func newResources() Resources {
	var resources Resources
	resources.Info.ServerAddress = getEnv("SERVER_ADDRESS", "localhost:8012")
	resources.Info.Environment = getEnv("ENV", "dev")
	resources.Info.DevEnvInstallation = getEnv("DEVENV_INSTALLATION", "")
	resources.Info.DatadogAPMService = getEnv("DD_SERVICE", "mcp-server")
	resources.Info.HAProxyURL = getEnv("HAPROXY_URL", "")

	if getEnv("DD_APM_TRACING_ENABLED", "false") == "true" {
		err := tracer.Start(
			tracer.WithAgentAddr(getEnv("DD_AGENT_HOST", "localhost")+":"+getEnv("DD_TRACE_AGENT_PORT", "8126")),
			tracer.WithDogstatsdAddr(getEnv("DD_AGENT_HOST", "localhost")+":"+getEnv("DD_DOGSTATSD_PORT", "8125")),
			tracer.WithEnv(getEnv("DD_ENV", resources.Info.Environment)),
			tracer.WithService(resources.Info.DatadogAPMService),
			tracer.WithServiceVersion(getEnv("DD_VERSION", "unknown")),
			tracer.WithGlobalTag("awsregion", getEnv("AWS_REGION", "unknown")),
			tracer.WithRuntimeMetrics(),
		)
		if err != nil {
			// the logger is not initialized yet, so we use the default logger
			slog.Default().Error("failed to start datadog tracer",
				slog.String("error", err.Error()),
			)
		}
	}

	return resources
}

// Logger returns the logger resource.
func (r *Resources) Logger() *slog.Logger {
	return r.logger
}

// IsDev returns true if the app is running in development environment.
func (r *Resources) IsDev() bool {
	return strings.EqualFold(r.Info.Environment, "dev")
}

// IsStaging returns true if the app is running in staging environment.
func (r *Resources) IsStaging() bool {
	return strings.EqualFold(r.Info.Environment, "staging")
}

// TeamworkHTTPClient returns the HTTP client to be used to make requests to
// Teamwork API.
func (r *Resources) TeamworkHTTPClient() *http.Client {
	return r.teamworkHTTPClient
}

// TeamworkEngine returns the Teamwork Engine instance to be used to make
// requests to Teamwork API.
func (r *Resources) TeamworkEngine() *twapi.Engine {
	return r.teamworkEngine
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
