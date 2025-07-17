package config

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/DataDog/dd-trace-go/v2/instrumentation/httptrace"
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
	resources.logger = slog.New(newCustomLogHandler(resources))
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
				return next.Do(req)
			})
		}),
		twapi.WithMiddleware(func(next twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
				// trace middleware
				if !resources.Info.DatadogAPM.Enabled {
					return next.Do(req)
				}
				_, ctx, finishSpans := httptrace.StartRequestSpan(req,
					tracer.Tag(ext.SpanKind, ext.SpanKindServer),
					tracer.Tag(ext.Component, "net/http"),
					tracer.ServiceName(resources.Info.DatadogAPM.Service),
					tracer.ResourceName(fmt.Sprintf("%s_%s", req.Method, req.URL.Path)),
					tracer.Tag(ext.HTTPRoute, req.Pattern),
				)
				req = req.WithContext(ctx)
				response, err := next.Do(req)
				finishSpans(response.StatusCode, nil, nil)
				return response, err
			})
		}),
		twapi.WithMiddleware(func(next twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
				if haProxyURL != nil {
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
func NewMCPServer(resources Resources, group *toolsets.ToolsetGroup) *server.MCPServer {
	mcpServer := server.NewMCPServer(mcpName, strings.TrimPrefix(resources.Info.Version, "v"),
		server.WithRecovery(),
		server.WithToolCapabilities(group.HasTools()),
		server.WithLogging(),
	)
	group.RegisterAll(mcpServer)
	return mcpServer
}
