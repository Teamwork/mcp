package config

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/getsentry/sentry-go"
	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/request"
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/session"
)

const (
	mcpName            = "Teamwork.com"
	sentryFlushTimeout = 2 * time.Second
)

// Load loads the configuration for the MCP service.
func Load() (Resources, func()) {
	resources := newResources()

	var haProxyURL *url.URL
	if resources.Info.HAProxyURL != "" {
		var err error
		if haProxyURL, err = url.Parse(resources.Info.HAProxyURL); err != nil {
			resources.logger.Error("failed to parse HAProxy URL",
				slog.String("url", resources.Info.HAProxyURL),
				slog.String("error", err.Error()),
			)
			// reset to nil to avoid using an invalid URL
			haProxyURL = nil
		}
	}

	resources.logger = slog.New(newCustomLogHandler(resources))

	resources.teamworkHTTPClient = new(http.Client)
	if haProxyURL != nil {
		// disable TLS verification when using HAProxy, as the certificate won't
		// match the internal address
		resources.teamworkHTTPClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(haProxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	resources.teamworkEngine = twapi.NewEngine(session.NewBearerTokenContext(),
		twapi.WithHTTPClient(resources.teamworkHTTPClient),
		twapi.WithMiddleware(func(next twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
				request.SetProxyHeaders(req)
				return next.Do(req)
			})
		}),
		twapi.WithLogger(resources.logger),
	)

	if resources.Info.DatadogAPM.Enabled {
		err := tracer.Start(
			tracer.WithAgentAddr(resources.Info.DatadogAPM.AgentHost+":"+resources.Info.DatadogAPM.AgentPort),
			tracer.WithDogstatsdAddr(resources.Info.DatadogAPM.AgentHost+":"+resources.Info.DatadogAPM.StatsdPort),
			tracer.WithEnv(resources.Info.DatadogAPM.Environment),
			tracer.WithService(resources.Info.DatadogAPM.Service),
			tracer.WithServiceVersion(resources.Info.DatadogAPM.Version),
			tracer.WithGlobalTag("awsregion", resources.Info.AWSRegion),
			tracer.WithRuntimeMetrics(),
		)
		if err != nil {
			// the logger is not initialized yet, so we use the default logger
			slog.Default().Error("failed to start datadog tracer",
				slog.String("error", err.Error()),
			)
		}
	}

	return resources, func() {
		if resources.Info.DatadogAPM.Enabled {
			tracer.Stop()
		}
		if resources.Info.Log.SentryDSN != "" {
			sentry.Flush(sentryFlushTimeout)
		}
	}
}

// NewMCPServer creates a new MCP server with the given resources and toolset
// group.
func NewMCPServer(resources Resources, group *toolsets.ToolsetGroup) *server.MCPServer {
	mcpServer := server.NewMCPServer(mcpName, strings.TrimPrefix(resources.Info.Version, "v"),
		server.WithRecovery(),
		server.WithToolCapabilities(group.HasTools()),
		server.WithLogging(),
	)
	group.RegisterAll(mcpServer)
	return mcpServer
}
