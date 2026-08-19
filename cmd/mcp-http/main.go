package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	ddhttp "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/getsentry/sentry-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/cli"
	"github.com/teamwork/mcp/internal/twchat"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
	"github.com/teamwork/mcp/pkg/auth"
	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/logsafe"
	"github.com/teamwork/mcp/pkg/request"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
	"github.com/teamwork/twapi-go-sdk/session"
)

var (
	reBearerToken = regexp.MustCompile(`^Bearer (.+)$`)
	methods       = cli.NewMethods(toolsets.MethodAll)
)

// Limit request body size (e.g., 10MB)
const maxBodySize = 10 * 1024 * 1024 // 10 MB

// openAIAppsChallengeToken proves to OpenAI that we control this origin, so the
// MCP server can be listed as a ChatGPT app. It is a public verification value,
// not a credential, and OpenAI expects it served verbatim as plain text from
// the origin root. Replace it if OpenAI issues a new token.
const openAIAppsChallengeToken = "qi_6vPrb4b2ob0EAizJTh0ziBCIjObAdxGkDegPy50Y"

func main() {
	defer handleExit()

	flag.Var(methods, "toolsets", "Comma-separated list of toolsets to enable")
	flag.Parse()

	resources, teardown := config.Load(os.Stdout, config.WithProfiles(methods.Profiles()...))
	defer teardown()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	groups, err := newToolsetGroups(resources)
	if err != nil {
		resources.Logger().Error("failed to create MCP server",
			slog.String("error", err.Error()),
		)
		exit(exitCodeSetupFailure)
	}
	mcpServer := config.NewMCPServer(resources, groups...)
	mcpHTTPServer := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless:                  true,
		DisableLocalhostProtection: resources.Info.Environment == "dev",
		// Pin the body limit to the one limitBodyMiddleware already enforces.
		// Left at zero the SDK applies its own DefaultMaxRequestBodyBytes (4 MiB),
		// which would silently tighten the limit clients have been coded against.
		MaxRequestBodyBytes: maxBodySize,
	})
	mcpSSEServer := mcp.NewSSEHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.SSEOptions{})

	mux := newRouter(resources, toolsets.Scopes(groups))
	mux.Handle("/sse", sseLogMiddleware(resources.Logger(), mcpSSEServer))
	mux.Handle("/", mcpHTTPServer)

	httpServer := &http.Server{
		Addr:    resources.Info.ServerAddress,
		Handler: addRouterMiddlewares(resources, mux),
	}

	resources.Logger().Info("starting http server",
		slog.String("address", resources.Info.ServerAddress),
	)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				resources.Logger().Error("failed to start server",
					slog.String("address", resources.Info.ServerAddress),
					slog.String("error", err.Error()),
				)
				select {
				case <-done:
				default:
					close(done)
				}
			}
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	if err := httpServer.Shutdown(ctx); err != nil {
		resources.Logger().Error("server shutdown failed",
			slog.String("error", err.Error()),
		)
	}
	resources.Logger().Info("server stopped")
}

// newToolsetGroups builds one ToolsetGroup per product. Each group declares its
// own tool prefix and OAuth scope, which is what both the tools/list scope
// filter and the advertised "scopes_supported" are derived from.
func newToolsetGroups(resources config.Resources) ([]*toolsets.ToolsetGroup, error) {
	projectsGroup := twprojects.DefaultToolsetGroup(false, false, resources.TeamworkEngine())
	if err := projectsGroup.EnableToolsets(methods.Toolsets()...); err != nil {
		return nil, fmt.Errorf("failed to enable toolsets: %w", err)
	}

	deskGroup := twdesk.DefaultToolsetGroup(false, resources.TeamworkHTTPClient())
	if err := deskGroup.EnableToolsets(methods.Toolsets()...); err != nil {
		return nil, fmt.Errorf("failed to enable desk toolsets: %w", err)
	}

	spacesGroup := twspaces.DefaultToolsetGroup(false, false, resources.TeamworkHTTPClient())
	if err := spacesGroup.EnableToolsets(methods.Toolsets()...); err != nil {
		return nil, fmt.Errorf("failed to enable spaces toolsets: %w", err)
	}

	chatGroup := twchat.DefaultToolsetGroup(false, resources.TeamworkEngine())
	if err := chatGroup.EnableToolsets(methods.Toolsets()...); err != nil {
		return nil, fmt.Errorf("failed to enable chat toolsets: %w", err)
	}

	return []*toolsets.ToolsetGroup{
		projectsGroup,
		deskGroup,
		spacesGroup,
		chatGroup,
	}, nil
}

func newRouter(resources config.Resources, scopes []string) *http.ServeMux {
	// Advertised verbatim from what the registered groups declare, so the scopes
	// a client may ask for always have a group behind them.
	scopesSupported, err := json.Marshal(scopes)
	if err != nil {
		// Marshalling a []string cannot fail, but advertising no scope beats
		// serving malformed metadata if it ever does.
		resources.Logger().Error("failed to encode supported scopes",
			slog.String("error", err.Error()),
		)
		scopesSupported = []byte("[]")
	}

	mux := http.NewServeMux()
	mux.Handle("/favicon.ico", http.RedirectHandler("https://teamwork.com/favicon.ico", http.StatusPermanentRedirect))
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodOptions {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) //nolint:errcheck
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodOptions {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodOptions {
			return
		}

		// https://datatracker.ietf.org/doc/html/rfc9728/#section-2
		_, _ = w.Write([]byte(`{
  "resource": "` + resources.Info.MCPURL + `",
  "authorization_servers": ["` + resources.Info.APIURL + `"],
  "bearer_methods_supported": ["header"],
  "resource_documentation": "https://apidocs.teamwork.com/guides/teamwork/app-login-flow",
  "scopes_supported": ` + string(scopesSupported) + `
}`))
	})
	mux.HandleFunc("/.well-known/openai-apps-challenge", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodOptions {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodOptions {
			return
		}

		_, _ = w.Write([]byte(openAIAppsChallengeToken))
	})
	return mux
}

func addRouterMiddlewares(resources config.Resources, mux *http.ServeMux) http.Handler {
	return chainMiddlewares(mux,
		func(h http.Handler) http.Handler { return stripProfile(resources.Info.MCPProfiles, h) },
		htmlIndexMiddleware,
		limitBodyMiddleware,
		requestInfoMiddleware,
		func(h http.Handler) http.Handler { return logMiddleware(resources.Logger(), h) },
		func(h http.Handler) http.Handler { return sentryMiddleware(resources, h) },
		func(h http.Handler) http.Handler { return tracerMiddleware(resources, h) },
		func(h http.Handler) http.Handler { return authMiddleware(resources, h) },
	)
}

// chainMiddlewares applies middlewares so the first argument is the outermost
// wrapper (runs first on the request, last on the response).
func chainMiddlewares(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// stripProfile is a middleware that checks if the request path starts with a
// known profile name, and if so, it strips the profile from the path and sets
// an "TW-MCP-Profile" header with the profile name. This allows clients to use
// URLs like "/project-manager/endpoint" to access the same endpoints as
// "/endpoint" but with a profile context.
func stripProfile(profiles []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the path starts with a known profile, strip it and set a header
		r.Header.Set("TW-MCP-Profile", "")
		for _, profile := range profiles {
			if strings.HasPrefix(r.URL.Path, "/"+profile+"/") {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, "/"+profile)
				r.Header.Add("TW-MCP-Profile", profile)
				break
			}
		}
		next.ServeHTTP(w, r)
	})
}

func htmlIndexMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the request is for the root path and accepts HTML, redirect to the MCP
		// homepage. This is a simplified check that doesn't cover Accept priority
		// (q values), but it should be sufficient to cover most cases (if not all).
		acceptHeader := r.Header.Get("Accept")
		isHTMLRquest := strings.Contains(acceptHeader, "text/html")
		if strings.Contains(acceptHeader, "application/json") || strings.Contains(acceptHeader, "text/event-stream") {
			isHTMLRquest = false
		}
		if r.URL.Path == "/" && isHTMLRquest {
			http.Redirect(w, r, "https://www.teamwork.com/ai/mcp/", http.StatusPermanentRedirect)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func limitBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		next.ServeHTTP(w, r)
	})
}

func requestInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(request.WithInfo(r.Context(), request.NewInfo(r))))
	})
}

func logMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	skipPaths := map[string]struct{}{
		"/favicon.ico": {}, // avoid logging browser favicon requests
		"/api/health":  {}, // health checks can be very noisy
		"/sse":         {}, // special log middleware
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, skip := skipPaths[r.URL.Path]; skip {
			next.ServeHTTP(w, r)
			return
		}

		var reqBody []byte
		if r.Body != nil {
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				logger.Error("failed to read request body", slog.String("error", err.Error()))
			}
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		start := time.Now()
		rw := request.NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		headers := r.Header.Clone()
		if auth := headers.Get("Authorization"); auth != "" {
			if authParts := strings.SplitN(auth, " ", 2); len(authParts) == 2 {
				headers.Set("Authorization", authParts[0]+" REDACTED")
			} else {
				headers.Set("Authorization", "REDACTED")
			}
		}

		info, _ := request.InfoFromContext(r.Context())
		logger.Info("request",
			slog.String("trace_id", info.TraceID()),
			slog.String("request_url", r.URL.String()),
			slog.String("request_method", r.Method),
			slog.Any("request_headers", headers),
			slog.String("request_body", logsafe.String(string(reqBody))),
			slog.Int("response_status", rw.StatusCode()),
			slog.Any("response_headers", rw.Header()),
			slog.String("response_body", string(rw.Body())),
			slog.Duration("duration", time.Since(start)),
			slog.Int64("installation.id", info.InstallationID()),
			slog.String("installation.url", info.InstallationURL()),
			slog.Int64("user.id", info.UserID()),
		)
	})
}

func sseLogMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := r.Header.Clone()
		if auth := headers.Get("Authorization"); auth != "" {
			if authParts := strings.SplitN(auth, " ", 2); len(authParts) == 2 {
				headers.Set("Authorization", authParts[0]+" REDACTED")
			} else {
				headers.Set("Authorization", "REDACTED")
			}
		}

		info, _ := request.InfoFromContext(r.Context())

		if r.Method == http.MethodGet {
			// long-lived SSE stream connection
			logger.Info("SSE stream opened",
				slog.String("trace_id", info.TraceID()),
				slog.String("request_url", r.URL.String()),
				slog.String("request_method", r.Method),
				slog.Any("request_headers", headers),
				slog.String("path", r.URL.String()),
			)
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("SSE stream closed",
				slog.String("trace_id", info.TraceID()),
				slog.Duration("duration", time.Since(start)),
			)
			return
		}

		// short-lived message deliveries to a session

		sessionID := r.URL.Query().Get("sessionid")

		var reqBody []byte
		if r.Body != nil {
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				logger.Error("failed to read SSE message body", slog.String("error", err.Error()))
			}
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		start := time.Now()
		rw := request.NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		logger.Info("SSE message",
			slog.String("trace_id", info.TraceID()),
			slog.String("session_id", sessionID),
			slog.String("request_url", r.URL.String()),
			slog.String("request_method", r.Method),
			slog.Any("request_headers", headers),
			slog.String("request_body", logsafe.String(string(reqBody))),
			slog.Int("response_status", rw.StatusCode()),
			slog.Any("response_headers", rw.Header()),
			slog.String("response_body", string(rw.Body())),
			slog.Duration("duration", time.Since(start)),
			slog.Int64("installation.id", info.InstallationID()),
			slog.String("installation.url", info.InstallationURL()),
			slog.Int64("user.id", info.UserID()),
		)
	})
}

func sentryMiddleware(resources config.Resources, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if resources.Info.Log.SentryDSN != "" {
			hub := sentry.CurrentHub().Clone()
			hub.Scope().SetRequest(r)
			ctx := sentry.SetHubOnContext(r.Context(), hub)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func tracerMiddleware(resources config.Resources, next http.Handler) http.Handler {
	if !resources.Info.DatadogAPM.Enabled {
		return next
	}
	skipPaths := map[string]struct{}{
		"/favicon.ico": {}, // avoid logging browser favicon requests
		"/api/health":  {}, // health checks can be very noisy
		"/sse":         {}, // long-lived connections don't work well with tracing
	}
	return ddhttp.WrapHandler(next, resources.Info.DatadogAPM.Service, "http.request",
		ddhttp.WithResourceNamer(func(req *http.Request) string {
			return fmt.Sprintf("%s_%s", req.Method, req.URL.Path)
		}),
		ddhttp.WithIgnoreRequest(func(req *http.Request) bool {
			if _, skip := skipPaths[req.URL.Path]; skip {
				return true
			}
			if strings.HasPrefix(req.URL.Path, "/.well-known") {
				return true
			}
			return false
		}),
	)
}

func authMiddleware(resources config.Resources, next http.Handler) http.Handler {
	whitelistEndpoints := map[string][]string{
		// health checks don't require authentication
		"/api/health": {http.MethodGet, http.MethodOptions},
		// browser may request favicons without authentication
		"/favicon.ico": {http.MethodGet, http.MethodOptions},
	}

	whitelistPrefixEndpoints := map[string][]string{
		// OAuth2 endpoints cannot require authentication
		"/.well-known": {"GET", "OPTIONS"},
	}

	validator := auth.NewValidator(resources.TeamworkHTTPClient(), resources.Info.APIURL, resources.Logger())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestLogger := resources.Logger().With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
		)

		if r.Header.Get("Authorization") == "" {
			// some endpoints don't require auth

			if methods, ok := whitelistEndpoints[r.URL.Path]; ok && slices.Contains(methods, r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			for prefix, methods := range whitelistPrefixEndpoints {
				if strings.HasPrefix(r.URL.Path, prefix) && slices.Contains(methods, r.Method) {
					next.ServeHTTP(w, r)
					return
				}
			}

			content, err := io.ReadAll(r.Body)
			if err != nil {
				requestLogger.ErrorContext(r.Context(), "failed to read request body",
					slog.String("error", err.Error()),
				)
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}

			bypass, err := auth.Bypass(content)
			switch {
			case err != nil, !bypass:
				// https://datatracker.ietf.org/doc/html/rfc9728#name-www-authenticate-response
				w.Header().Set("WWW-Authenticate",
					`Bearer resource_metadata="`+resources.Info.MCPURL+`/.well-known/oauth-protected-resource"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			default:
				r.Body = io.NopCloser(bytes.NewBuffer(content))
				next.ServeHTTP(w, r)
			}
			return
		}

		matches := reBearerToken.FindStringSubmatch(r.Header.Get("Authorization"))
		if len(matches) < 2 {
			// https://datatracker.ietf.org/doc/html/rfc9728#name-www-authenticate-response
			w.Header().Set("WWW-Authenticate",
				`Bearer resource_metadata="`+resources.Info.MCPURL+`/.well-known/oauth-protected-resource"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		bearerToken := matches[1]

		info, err := validator.GetBearerInfo(r.Context(), bearerToken)
		switch {
		case errors.Is(err, auth.ErrBearerInfoUnauthorized):
			// The token was positively rejected, so challenge the client to
			// re-authorise.
			// https://datatracker.ietf.org/doc/html/rfc9728#name-www-authenticate-response
			w.Header().Set("WWW-Authenticate",
				`Bearer resource_metadata="`+resources.Info.MCPURL+`/.well-known/oauth-protected-resource"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return

		case errors.Is(err, auth.ErrBearerInfoCanceled):
			// The client hung up while we were validating. Nobody is left to
			// read a response and nothing failed on our side, so this is debug
			// noise rather than an error.
			requestLogger.DebugContext(r.Context(), "bearer info validation canceled by client",
				slog.String("error", err.Error()),
			)
			return

		case errors.Is(err, auth.ErrBearerInfoUnavailable):
			// We could not determine whether the token is valid. Never send a
			// re-authorisation challenge here: the token is probably fine, and
			// telling the client to discard it puts the user through the whole
			// OAuth flow (and MFA) over what may be a momentary blip.
			requestLogger.ErrorContext(r.Context(), "failed to validate bearer info",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to validate bearer token", http.StatusServiceUnavailable)
			return

		case err != nil:
			requestLogger.ErrorContext(r.Context(), "failed to get bearer info",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to get bearer info", http.StatusInternalServerError)
			return
		}

		if span, ok := tracer.SpanFromContext(r.Context()); ok {
			span.SetTag("user.id", info.UserID)
			span.SetTag("installation.id", info.InstallationID)
			span.SetTag("installation.url", info.URL)
		}
		if requestInfo, ok := request.InfoFromContext(r.Context()); ok {
			requestInfo.SetAuth(info.InstallationID, info.URL, info.UserID)
		}

		ctx := r.Context()
		// detect cross-region requests
		ctx = twctx.WithCrossRegion(ctx, !strings.EqualFold(resources.Info.AWSRegion, info.Region))
		// inject customer URL
		ctx = twctx.WithCustomerURL(ctx, info.URL)
		// inject bearer token
		ctx = twctx.WithBearerToken(ctx, bearerToken)
		// inject scopes
		ctx = twctx.WithScopes(ctx, info.Meta.Scopes)
		// inject session
		ctx = session.WithBearerTokenContext(ctx, session.NewBearerToken(bearerToken, info.URL))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type exitCode int

const (
	exitCodeOK exitCode = iota
	exitCodeSetupFailure
)

type exitData struct {
	code exitCode
}

// exit allows to abort the program while still executing all defer statements.
func exit(code exitCode) {
	panic(exitData{code: code})
}

// handleExit exit code handler.
func handleExit() {
	if e := recover(); e != nil {
		if exit, ok := e.(exitData); ok {
			os.Exit(int(exit.code))
		}
		panic(e)
	}
}
