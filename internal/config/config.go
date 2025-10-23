package config

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	ddhttp "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/DataDog/dd-trace-go/v2/instrumentation/httptrace"
	"github.com/getsentry/sentry-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
func NewMCPServer(resources Resources, groups ...*toolsets.ToolsetGroup) *mcp.Server {
	// Determine if any group has tools
	hasTools := false
	for _, group := range groups {
		if group.HasTools() {
			hasTools = true
			break
		}
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    mcpName,
		Title:   "Teamwork.com Model Context Protocol",
		Version: strings.TrimPrefix(resources.Info.Version, "v"),
	}, &mcp.ServerOptions{
		HasTools: hasTools,
	})
	mcpServer.AddSendingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			result, err = next(ctx, method, req)
			if err != nil {
				return result, err
			}

			// populate Datadog APM trace with more information about the MCP
			// request/response
			if resources.Info.DatadogAPM.Enabled {
				if span, ok := tracer.SpanFromContext(ctx); ok {
					span.SetTag("mcp.method", method)
					if callToolParams, ok := req.GetParams().(*mcp.CallToolParams); ok {
						span.SetTag("mcp.tool.name", callToolParams.Name)
						if encoded, err := json.Marshal(callToolParams.Arguments); err == nil {
							span.SetTag("mcp.tool.arguments", string(encoded))
						}
					}
					if callToolResult, ok := result.(*mcp.CallToolResult); ok {
						if callToolResult.IsError {
							if encoded, err := json.Marshal(callToolResult.Content); err == nil {
								span.SetTag(ext.Error, encoded)
							} else {
								span.SetTag(ext.Error, "failed to execute tool")
							}
						}
					}
				}
			}

			listToolsResult, ok := result.(*mcp.ListToolsResult)
			if !ok || listToolsResult == nil || len(listToolsResult.Tools) == 0 {
				return result, nil
			}

			// filter tools based on scopes
			scopes := scopes(ctx)
			if len(scopes) == 0 {
				return
			}

			projectsScope := slices.Contains(scopes, "projects")
			deskScope := slices.Contains(scopes, "desk")

			listToolsResult.Tools = slices.DeleteFunc(listToolsResult.Tools, func(tool *mcp.Tool) bool {
				return (strings.HasPrefix(tool.Name, "twprojects") && !projectsScope) ||
					(strings.HasPrefix(tool.Name, "twdesk") && !deskScope)
			})
			return listToolsResult, nil
		}
	})

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
	transport mcp.Transport,
	options *mcp.ClientOptions,
) (*mcp.Client, *mcp.ClientSession, error) {
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    mcpName,
		Title:   "Teamwork.com Model Context Protocol",
		Version: strings.TrimPrefix(resources.Info.Version, "v"),
	}, options)

	clientSession, err := mcpClient.Connect(ctx, transport, &mcp.ClientSessionOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	return mcpClient, clientSession, nil
}
