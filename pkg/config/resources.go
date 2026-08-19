package config

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	desksdk "github.com/teamwork/desksdkgo/client"
	twapi "github.com/teamwork/twapi-go-sdk"
)

const (
	// defaultEnvPrefix is the prefix every configuration variable is read under
	// unless WithEnvPrefix says otherwise.
	defaultEnvPrefix = "TW_MCP_"

	defaultServerName  = "Teamwork.com"
	defaultServerTitle = "Teamwork.com Model Context Protocol"
	defaultMCPURL      = "https://mcp.ai.teamwork.com"
)

// Version is the current version of the MCP server. It is set at build time
// using -ldflags "-X 'github.com/teamwork/mcp/pkg/config.Version=1.0.0'".
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
		// Name is the MCP server name reported in the initialize handshake.
		Name string
		// Title is the human-readable MCP server title reported in the initialize
		// handshake.
		Title string
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
		// MCPProfiles is the list of profiles to be enabled in the MCP server. This
		// is useful for the MCP server in HTTP mode, where different profiles can
		// be accessed via different URL paths (e.g. "/project-manager/endpoint" for
		// the "project-manager" profile).
		MCPProfiles []string
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

// Option adjusts what Load builds. A server built on this package uses these to
// identify itself and to read its own environment, rather than inheriting this
// server's name and TW_MCP_ variables.
type Option func(*options)

type options struct {
	envPrefix string
	name      string
	title     string
	mcpURL    string
	profiles  []string
}

// WithEnvPrefix sets the prefix every configuration variable is read under.
// Defaults to "TW_MCP_", so TW_MCP_SERVER_ADDRESS and friends. A second server
// sharing a deployment needs its own prefix to be configured independently.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) { o.envPrefix = prefix }
}

// WithServerIdentity sets the name and title the MCP server reports in the
// initialize handshake. Clients key their connector UI off these, so two
// servers must not share them.
func WithServerIdentity(name, title string) Option {
	return func(o *options) {
		o.name = name
		o.title = title
	}
}

// WithDefaultMCPURL sets the base URL this server reports as its own resource
// identifier when the environment does not say. It must identify *this* server:
// it is the "resource" in the RFC 9728 protected-resource metadata and the
// resource_metadata pointer in every 401 challenge, so a server left on another
// server's URL sends clients to authorise against the wrong resource.
//
// The environment variable still wins, so a deployment can override it.
func WithDefaultMCPURL(url string) Option {
	return func(o *options) { o.mcpURL = url }
}

// WithProfiles sets the named toolset profiles this server exposes as URL path
// prefixes.
func WithProfiles(profiles ...string) Option {
	return func(o *options) { o.profiles = profiles }
}

func newOptions(opts ...Option) options {
	resolved := options{
		envPrefix: defaultEnvPrefix,
		name:      defaultServerName,
		title:     defaultServerTitle,
		mcpURL:    defaultMCPURL,
	}
	for _, opt := range opts {
		opt(&resolved)
	}
	return resolved
}

func newResources(opts options) Resources {
	// env reads this server's own configuration, under its prefix. The OTel
	// variables below deliberately use the bare getEnv: those names come from the
	// OpenTelemetry specification, not from this server.
	env := func(key, fallback string) string {
		return getEnvWithPrefix(opts.envPrefix, key, fallback)
	}
	profiles := opts.profiles

	var resources Resources
	resources.Info.Name = env("NAME", opts.name)
	resources.Info.Title = env("TITLE", opts.title)
	resources.Info.Version = env("VERSION", Version)
	resources.Info.ServerAddress = env("SERVER_ADDRESS", ":8080")
	resources.Info.Environment = env("ENV", "dev")
	resources.Info.AWSRegion = env("AWS_REGION", "us-east-1")
	resources.Info.MCPURL = strings.TrimSuffix(env("URL", opts.mcpURL), "/")
	resources.Info.MCPProfiles = profiles
	resources.Info.APIURL = strings.TrimSuffix(env("API_URL", "https://teamwork.com"), "/")
	resources.Info.HAProxyURL = env("HAPROXY_URL", "")
	resources.Info.BearerToken = env("BEARER_TOKEN", "")
	resources.Info.Log.Format = strings.ToLower(env("LOG_FORMAT", "text"))
	resources.Info.Log.Level = strings.ToLower(env("LOG_LEVEL", "info"))
	resources.Info.Log.SentryDSN = env("SENTRY_DSN", "")

	// https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/
	resources.Info.OTel.Enabled = strings.EqualFold(getEnv("OTEL_TRACING_ENABLED", "false"), "true")
	resources.Info.OTel.Endpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	resources.Info.OTel.Service = getEnv("OTEL_SERVICE_NAME", "mcp-server")
	resources.Info.OTel.Environment = getEnv("OTEL_ENV", resources.Info.Environment)
	resources.Info.OTel.Version = getEnv("OTEL_VERSION", resources.Info.Version)

	// only append the profile to the MCP URL if there is exactly one profile, to
	// avoid confusion with multiple profiles
	var mcpURLHasProfile bool
	for _, profile := range profiles {
		if strings.HasSuffix(resources.Info.MCPURL, "/"+profile) {
			mcpURLHasProfile = true
			break
		}
	}
	if len(profiles) == 1 && !mcpURLHasProfile {
		resources.Info.MCPURL += "/" + profiles[0]
	}

	return resources
}

// Logger returns the logger resource. A Resources that did not come from Load
// carries none, so callers get the default logger rather than dereferencing
// nil — a server built on this package may well assemble one by hand.
func (r *Resources) Logger() *slog.Logger {
	if r.logger == nil {
		return slog.Default()
	}
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

func getEnvWithPrefix(prefix, key, fallback string) string {
	if value, ok := os.LookupEnv(prefix + key); ok {
		return value
	}
	return fallback
}

// getEnv reads an unprefixed variable. Only the OTel variables use it: those
// names are set by the OpenTelemetry specification, not by this server.
func getEnv(key, fallback string) string {
	return getEnvWithPrefix("", key, fallback)
}
