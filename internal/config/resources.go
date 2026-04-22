package config

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	desksdk "github.com/teamwork/desksdkgo/client"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// Version is the current version of the MCP server. It is set at build time
// using -ldflags "-X 'github.com/teamwork/mcp/internal/config.Version=1.0.0'".
// If not set, it defaults to "dev".
var Version = "dev"

// Resources stores all the resources loaded in the startup.
type Resources struct {
	teamworkHTTPClient *http.Client
	teamworkEngine     *twapi.Engine
	deskClient         *desksdk.Client
	logger             *slog.Logger

	// Info stores environment variables mappings.
	Info struct {
		// Version is the current version of the MCP server.
		Version string
		// ServerAddress is the address of the server. This is useful for the MCP
		// server in HTTP mode.
		ServerAddress string
		// Environment is the environment this app is running in.
		Environment string
		// AWSRegion is the AWS region this app is running in.
		AWSRegion string
		// MCPURL is the base URL of the MCP server. This is useful for the MCP
		// server in HTTP mode.
		MCPURL string
		// APIURL is the base URL of the Teamwork API.
		APIURL string
		// HAProxyURL is the URL of the HAProxy instance. This is useful for the MCP
		// server in HTTP mode.
		HAProxyURL string
		// BearerToken is the bearer token to be used to authenticate with Teamwork
		// API. This is useful for the MCP server in STDIO mode.
		BearerToken string
		// Log contains the logging configuration.
		Log struct {
			// Format is the format of the logs. It can be "json" or "text".
			Format string
			// Level is the log level. It can be "debug", "info", "warn", "error" or
			// "fatal".
			Level string
			// SentryDSN is the Sentry DSN to be used for error reporting.
			SentryDSN string
		}
		// OTel contains the configuration for OpenTelemetry tracing.
		OTel struct {
			// Enabled indicates if OpenTelemetry tracing is enabled.
			Enabled bool
			// Endpoint is the OTLP HTTP endpoint to send traces to (e.g. "http://localhost:4318").
			Endpoint string
			// Service is the name of the service.
			Service string
			// Environment is the deployment environment (e.g. "production", "staging").
			Environment string
			// Version is the version of the service.
			Version string
		}
	}
}

func newResources() Resources {
	var resources Resources
	resources.Info.Version = getEnv("TW_MCP_VERSION", Version)
	resources.Info.ServerAddress = getEnv("TW_MCP_SERVER_ADDRESS", ":8080")
	resources.Info.Environment = getEnv("TW_MCP_ENV", "dev")
	resources.Info.AWSRegion = getEnv("TW_MCP_AWS_REGION", "us-east-1")
	resources.Info.MCPURL = strings.TrimSuffix(getEnv("TW_MCP_URL", "https://mcp.ai.teamwork.com"), "/")
	resources.Info.APIURL = strings.TrimSuffix(getEnv("TW_MCP_API_URL", "https://teamwork.com"), "/")
	resources.Info.HAProxyURL = getEnv("TW_MCP_HAPROXY_URL", "")
	resources.Info.BearerToken = getEnv("TW_MCP_BEARER_TOKEN", "")
	resources.Info.Log.Format = strings.ToLower(getEnv("TW_MCP_LOG_FORMAT", "text"))
	resources.Info.Log.Level = strings.ToLower(getEnv("TW_MCP_LOG_LEVEL", "info"))
	resources.Info.Log.SentryDSN = getEnv("TW_MCP_SENTRY_DSN", "")

	resources.Info.OTel.Enabled = strings.EqualFold(getEnv("OTEL_TRACING_ENABLED", "false"), "true")
	resources.Info.OTel.Endpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	resources.Info.OTel.Service = getEnv("OTEL_SERVICE_NAME", "mcp-server")
	resources.Info.OTel.Environment = getEnv("OTEL_ENV", resources.Info.Environment)
	resources.Info.OTel.Version = getEnv("OTEL_VERSION", resources.Info.Version)

	return resources
}

// Logger returns the logger resource.
func (r *Resources) Logger() *slog.Logger {
	return r.logger
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

// DeskClient returns the Teamwork Desk Client for use.
func (r *Resources) DeskClient() *desksdk.Client {
	return r.deskClient
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
