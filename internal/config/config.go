package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	ddhttp "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/DataDog/dd-trace-go/v2/instrumentation/httptrace"
	"github.com/getsentry/sentry-go"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	desksdk "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/internal/network"
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
func Load(logOutput io.Writer) (Resources, func()) {
	resources := newResources()
	resources.logger = slog.New(newCustomLogHandler(resources, logOutput))
	resources.teamworkHTTPClient = new(http.Client)

	var haProxyURL *url.URL
	if resources.Info.HAProxyURL != "" {
		var err error
		if haProxyURL, err = url.Parse(resources.Info.HAProxyURL); err != nil {
			resources.logger.Error("failed to parse HAProxy URL",
				slog.String("url", resources.Info.HAProxyURL),
				slog.String("error", err.Error()),
			)
			haProxyURL = nil

		} else {
			// disable TLS verification when using HAProxy, as the certificate won't
			// match the internal address
			resources.teamworkHTTPClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}

			resources.logger.Info("using HAProxy for Teamwork API requests",
				slog.String("url", haProxyURL.String()),
			)
		}
	}

	if resources.Info.DatadogAPM.Enabled {
		resources.teamworkHTTPClient = ddhttp.WrapClient(resources.teamworkHTTPClient,
			ddhttp.WithService(resources.Info.DatadogAPM.Service),
			ddhttp.WithResourceNamer(func(req *http.Request) string {
				return fmt.Sprintf("%s_%s", req.Method, req.URL.Path)
			}),
			ddhttp.WithBefore(func(r *http.Request, s *tracer.Span) {
				// update the span URL when using internal HAProxy address
				if host := r.Header.Get("Host"); host != "" && host != r.URL.Host {
					url := httptrace.URLFromRequest(r, true)
					url = strings.Replace(url, r.URL.Host, host, 1)
					s.SetTag(ext.HTTPURL, url)
				}
			}),
		)
	}

	// Allow logging HTTP requests
	resources.teamworkHTTPClient.Transport = network.NewLoggingRoundTripper(
		resources.logger,
		resources.teamworkHTTPClient.Transport,
	)

	resources.teamworkEngine = twapi.NewEngine(session.NewBearerTokenContext(),
		twapi.WithHTTPClient(resources.teamworkHTTPClient),
		twapi.WithMiddleware(func(next twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
				// add request information to Sentry reports
				if resources.Info.Log.SentryDSN != "" {
					hub := sentry.CurrentHub().Clone()
					hub.Scope().SetRequest(req)
					ctx := sentry.SetHubOnContext(req.Context(), hub)
					req = req.WithContext(ctx)
				}
				return next.Do(req)
			})
		}),
		twapi.WithMiddleware(func(next twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
				// add proxy headers
				request.SetProxyHeaders(req)
				// add user agent
				req.Header.Set("User-Agent", "Teamwork MCP/"+resources.Info.Version)
				return next.Do(req)
			})
		}),
		twapi.WithMiddleware(func(next twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
				if haProxyURL != nil && !isCrossRegion(req.Context()) {
					// use internal HAProxy address to avoid extra hops
					req.Header.Set("Host", req.URL.Host)
					req.URL.Host = haProxyURL.Host
					req.URL.Scheme = haProxyURL.Scheme
				}
				return next.Do(req)
			})
		}),
		twapi.WithLogger(resources.logger),
	)

	resources.deskClient = desksdk.NewClient(
		resources.Info.APIURL+"/desk/api/v2",
		desksdk.WithHTTPClient(resources.teamworkHTTPClient),
		desksdk.WithMiddleware(
			func(
				ctx context.Context,
				req *http.Request,
				next desksdk.RequestHandler,
			) (*http.Response, error) {
				// Get the bearer token from the context (if available)
				btx := session.NewBearerTokenContext()
				err := btx.Authenticate(ctx, req)
				if err != nil {
					return nil, err
				}

				request.SetProxyHeaders(req)
				req.Header.Set("User-Agent", "Teamwork MCP/"+resources.Info.Version)
				return next(ctx, req)
			}),
	)

	if resources.Info.DatadogAPM.Enabled {
		if err := startDatadog(resources); err != nil {
			resources.logger.Error("failed to start datadog tracer",
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
func NewMCPServer(resources Resources, groups ...*toolsets.ToolsetGroup) *server.MCPServer {
	// Determine if any group has tools
	hasTools := false
	for _, group := range groups {
		if group.HasTools() {
			hasTools = true
			break
		}
	}

	mcpServer := server.NewMCPServer(mcpName, strings.TrimPrefix(resources.Info.Version, "v"),
		server.WithRecovery(),
		server.WithToolCapabilities(hasTools),
		server.WithLogging(),
	)

	// Register all toolset groups
	for _, group := range groups {
		group.RegisterAll(mcpServer)
	}

	return mcpServer
}

// NewMCPClient creates a new MCP client.
func NewMCPClient(
	ctx context.Context,
	resources Resources,
	transport transport.Interface,
	options ...client.ClientOption,
) (*client.Client, *mcp.InitializeResult, error) {
	mcpClient := client.NewClient(transport, options...)

	mcpClient.OnNotification(func(notification mcp.JSONRPCNotification) {
		resources.logger.Info("MCP notification",
			slog.String("method", notification.Method),
		)
	})

	mcpServerInfo, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    mcpName,
				Version: strings.TrimPrefix(resources.Info.Version, "v"),
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}
	return mcpClient, mcpServerInfo, nil
}
