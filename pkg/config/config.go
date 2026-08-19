package config

import (
	"cmp"
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

	"github.com/getsentry/sentry-go"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	desksdk "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/pkg/logsafe"
	"github.com/teamwork/mcp/pkg/network"
	"github.com/teamwork/mcp/pkg/presigned"
	"github.com/teamwork/mcp/pkg/request"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/session"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	otelattr "go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	sentryFlushTimeout = 2 * time.Second

	// cacheScopePrivate marks a cacheable result as only cacheable by the
	// requesting user's client, never by a shared intermediary. See SEP-2549 and
	// the mcp.Cacheable type, whose CacheScope field this fills in.
	cacheScopePrivate = "private"

	// keepaliveInterval is how often an idle peer is pinged, and
	// keepaliveFailureThreshold how many consecutive misses close the session.
	// The SDK closes on the first miss, which turns a single dropped ping into a
	// dead session; the spec expects multiple failures before that.
	keepaliveInterval         = 30 * time.Second
	keepaliveFailureThreshold = 3

	// protocolVersionWithoutPing is the first protocol version that removed the
	// "ping" method (SEP-2577). See keepalivePingGate.
	protocolVersionWithoutPing = "2026-07-28"

	// namespaceSeparator divides a tool's namespace from its action, as in
	// "twprojects-get_task". See namespaceTable.allows.
	namespaceSeparator = "-"
)

// Load loads the configuration for the MCP service. Options let a server built
// on this package supply its own identity, environment-variable prefix and
// toolset profiles; without them it loads this server's defaults.
func Load(logOutput io.Writer, opts ...Option) (Resources, func()) {
	resources := newResources(newOptions(opts...))
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
			// match the internal address. Pre-signed storage uploads keep it: they
			// are not rerouted, so their certificate does match.
			resources.teamworkHTTPClient.Transport = &network.PresignedSplitTransport{
				Base: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}

			resources.logger.Info("using HAProxy for Teamwork API requests",
				slog.String("url", haProxyURL.String()),
			)
		}
	}

	if resources.Info.OTel.Enabled {
		transport := resources.teamworkHTTPClient.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		resources.teamworkHTTPClient.Transport = otelhttp.NewTransport(transport,
			otelhttp.WithSpanNameFormatter(func(_ string, req *http.Request) string {
				return fmt.Sprintf("%s_%s", req.Method, req.URL.Path)
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
				// A pre-signed URL addresses storage, not the API, and its signature
				// covers the host, so rerouting it sends the file to the wrong server
				// with a signature that cannot match.
				if haProxyURL != nil && !twctx.IsCrossRegion(req.Context()) && !presigned.IsURL(req.URL) {
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

	var otelShutdown func(context.Context) error
	if resources.Info.OTel.Enabled {
		var err error
		otelShutdown, err = startOTel(context.Background(), resources)
		if err != nil {
			resources.logger.Error("failed to start OpenTelemetry tracer",
				slog.String("error", err.Error()),
			)
		}
	}

	return resources, func() {
		if otelShutdown != nil {
			if err := otelShutdown(context.Background()); err != nil {
				resources.logger.Error("failed to shutdown OpenTelemetry tracer",
					slog.String("error", err.Error()),
				)
			}
		}
		if resources.Info.Log.SentryDSN != "" {
			sentry.Flush(sentryFlushTimeout)
		}
	}
}

// NewMCPServer creates a new MCP server with the given resources and toolset
// group.
func NewMCPServer(resources Resources, groups ...*toolsets.ToolsetGroup) *mcp.Server {
	// Determine if any group has tools, prompts or resources to populate the
	// server capabilities
	var hasTools, hasPrompts, hasResources bool
	for _, group := range groups {
		if group.HasTools() {
			hasTools = true
		}
		if group.HasPrompts() {
			hasPrompts = true
		}
		if group.HasResources() {
			hasResources = true
		}
	}

	serverOptions := &mcp.ServerOptions{
		Logger:     resources.logger,
		HasTools:   hasTools,
		HasPrompts: hasPrompts,
		// The "logging" capability is deliberately not advertised. It was
		// deprecated by SEP-2577, and this server never sent a
		// "notifications/message" anyway, so claiming it only told clients to call
		// "logging/setLevel" for no effect. Old clients are unaffected: the SDK
		// still answers "logging/setLevel", and auth.Bypass still whitelists it.
		Capabilities: &mcp.ServerCapabilities{
			Extensions: map[string]any{
				// https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/2026-01-26/apps.mdx#extension-identifier
				"io.modelcontextprotocol/ui": map[string]any{},
			},
		},
		// https://github.com/modelcontextprotocol/go-sdk/blob/v1.5.0/design/design.md#ping--keepalive
		KeepAlive:                 keepaliveInterval,
		KeepAliveFailureThreshold: keepaliveFailureThreshold,
	}
	if hasTools {
		serverOptions.Capabilities.Tools = &mcp.ToolCapabilities{}
	}
	if hasPrompts {
		serverOptions.Capabilities.Prompts = &mcp.PromptCapabilities{}
	}
	if hasResources {
		serverOptions.Capabilities.Resources = &mcp.ResourceCapabilities{}
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    resources.Info.Name,
		Title:   resources.Info.Title,
		Version: strings.TrimPrefix(resources.Info.Version, "v"),
	}, serverOptions)
	namespaces := newNamespaceTable(groups)

	mcpServer.AddReceivingMiddleware(mcpLoggingMiddleware(resources))
	mcpServer.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			result, err = next(ctx, method, req)
			if err != nil {
				return result, err
			}

			// populate OTel trace with MCP information
			if resources.Info.OTel.Enabled {
				span := oteltrace.SpanFromContext(ctx)
				span.SetAttributes(otelattr.String("mcp.method", method))
				if callToolParams, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok {
					span.SetAttributes(
						otelattr.String("mcp.tool.name", callToolParams.Name),
						otelattr.String("mcp.tool.arguments", logsafe.String(string(callToolParams.Arguments))),
					)
				}
				if callToolResult, ok := result.(*mcp.CallToolResult); ok {
					if callToolResult.IsError {
						if encoded, encErr := json.Marshal(callToolResult.Content); encErr == nil {
							span.SetStatus(otelcodes.Error, string(encoded))
						} else {
							span.SetStatus(otelcodes.Error, "failed to execute tool")
						}
					}
				}
			}

			listToolsResult, ok := result.(*mcp.ListToolsResult)
			if !ok || listToolsResult == nil {
				return result, nil
			}

			// The tool list is filtered per OAuth token scopes below, so it is
			// specific to the requesting user. The SDK defaults cacheable results
			// to "public" (SEP-2549), which would allow a shared intermediary to
			// serve one tenant's tool list to another. Downgrade it to "private".
			listToolsResult.CacheScope = cacheScopePrivate

			if len(listToolsResult.Tools) == 0 {
				return listToolsResult, nil
			}

			// filter tools based on scopes
			if tokenScopes := twctx.ScopesFromContext(ctx); len(tokenScopes) > 0 {
				listToolsResult.Tools = slices.DeleteFunc(listToolsResult.Tools, func(tool *mcp.Tool) bool {
					return !namespaces.allows(tool.Name, tokenScopes)
				})
			}

			// order tools so the most commonly used appear first. This helps MCP
			// clients that truncate the tool list at a fixed size to keep the most
			// useful tools. Tools not in the preferred list follow alphabetically.
			orderTools(listToolsResult.Tools)
			return listToolsResult, nil
		}
	})

	mcpServer.AddSendingMiddleware(keepalivePingGate())

	// Register all toolset groups
	for _, group := range groups {
		group.RegisterAll(mcpServer)
	}

	return mcpServer
}

// namespaceTable maps a tool-name prefix to the OAuth scope that grants access
// to it, built from what each ToolsetGroup declares via SetNamespace. Driving
// the tools/list filter from this rather than a hardcoded list of products is
// what lets a server built on this package add its own groups without editing
// the filter — a group whose scope the filter did not know about would
// otherwise be listed to every token.
type namespaceTable []struct {
	toolPrefix string
	scope      string
}

func newNamespaceTable(groups []*toolsets.ToolsetGroup) namespaceTable {
	var table namespaceTable
	for _, group := range groups {
		prefix, scope := group.ToolPrefix(), group.Scope()
		if prefix == "" || scope == "" {
			continue
		}
		table = append(table, struct {
			toolPrefix string
			scope      string
		}{toolPrefix: prefix, scope: scope})
	}
	return table
}

// allows reports whether a token carrying the given scopes may see the named
// tool. A tool matching no known prefix is always allowed: the table only
// describes the groups that opted into scoping, so an unprefixed tool is not
// something this filter can decide on.
//
// The match is on the whole "<prefix>-" segment rather than a raw string
// prefix, because namespaces are free to be prefixes of one another — a
// "twpro" namespace would otherwise swallow every "twprojects-" tool and hand
// it out under the wrong scope.
func (n namespaceTable) allows(toolName string, tokenScopes []string) bool {
	for _, namespace := range n {
		if strings.HasPrefix(toolName, namespace.toolPrefix+namespaceSeparator) {
			return slices.Contains(tokenScopes, namespace.scope)
		}
	}
	return true
}

// keepalivePingGate stops the keepalive from sending "ping" to a peer whose
// negotiated protocol version no longer has the method. The SDK starts the
// keepalive at connect time, before any version is known, and never revisits
// that decision, so the gate has to sit on the sending path.
//
// MethodNotFound is what the keepalive loop reads as "peer does not support
// ping": it stops pinging and leaves the session open. That is the same outcome
// as a compliant peer rejecting the request, minus the round trip and the error
// it leaves in the client's log.
func keepalivePingGate() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "ping" {
				return next(ctx, method, req)
			}
			session, ok := req.GetSession().(*mcp.ServerSession)
			if !ok {
				return next(ctx, method, req)
			}
			// Before the handshake there is no version to read, and the spec allows a
			// ping there, so let it through.
			params := session.InitializeParams()
			if params == nil || params.ProtocolVersion < protocolVersionWithoutPing {
				return next(ctx, method, req)
			}
			return nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeMethodNotFound,
				Message: fmt.Sprintf("ping was removed in protocol version %s", protocolVersionWithoutPing),
			}
		}
	}
}

// orderTools sorts tools in place so that those registered via
// toolsets.RegisterToolOrder appear first (in that order), followed by the
// remaining tools alphabetically by name. This helps MCP clients that truncate
// the tool list at a fixed size keep the most useful tools.
func orderTools(tools []*mcp.Tool) {
	order := toolsets.ToolOrder()
	ranks := make(map[string]int, len(order))
	for i, method := range order {
		ranks[method.String()] = i
	}

	slices.SortStableFunc(tools, func(a, b *mcp.Tool) int {
		rankA, okA := ranks[a.Name]
		rankB, okB := ranks[b.Name]
		switch {
		case okA && okB:
			return cmp.Compare(rankA, rankB)
		case okA:
			return -1
		case okB:
			return 1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})
}

// NewMCPClient creates a new MCP client.
func NewMCPClient(
	ctx context.Context,
	resources Resources,
	transport mcp.Transport,
	options *mcp.ClientOptions,
) (*mcp.Client, *mcp.ClientSession, error) {
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    resources.Info.Name,
		Title:   resources.Info.Title,
		Version: strings.TrimPrefix(resources.Info.Version, "v"),
	}, options)

	clientSession, err := mcpClient.Connect(ctx, transport, &mcp.ClientSessionOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	return mcpClient, clientSession, nil
}

func mcpLoggingMiddleware(resources Resources) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			logger := resources.Logger()

			info, _ := request.InfoFromContext(ctx)
			attrs := []any{
				slog.String("mcp.method", method),
				slog.String("trace_id", info.TraceID()),
				slog.Int64("installation.id", info.InstallationID()),
				slog.String("installation.url", info.InstallationURL()),
				slog.Int64("user.id", info.UserID()),
			}

			if params, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok {
				attrs = append(attrs,
					slog.String("mcp.tool.name", params.Name),
					slog.String("mcp.tool.arguments", logsafe.String(string(params.Arguments))),
				)
			}

			start := time.Now()
			result, err := next(ctx, method, req)
			duration := time.Since(start)

			attrs = append(attrs, slog.Duration("mcp.duration", duration))

			if err != nil {
				attrs = append(attrs, slog.String("mcp.error", err.Error()))
				logger.Error("MCP request failed", attrs...)
				return result, err
			}

			if callToolResult, ok := result.(*mcp.CallToolResult); ok {
				attrs = append(attrs, slog.Bool("mcp.tool.is_error", callToolResult.IsError))
				if callToolResult.IsError {
					if encoded, encErr := json.Marshal(callToolResult.Content); encErr == nil {
						attrs = append(attrs, slog.String("mcp.tool.error_content", string(encoded)))
					}
				}
			}

			logger.Info("MCP request", attrs...)
			return result, nil
		}
	}
}
