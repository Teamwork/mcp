package main

import (
	"bytes"
	"context"
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
	"github.com/teamwork/mcp/internal/auth"
	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/request"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk/session"
)

var reBearerToken = regexp.MustCompile(`^Bearer (.+)$`)

// Limit request body size (e.g., 10MB)
const maxBodySize = 10 * 1024 * 1024 // 10 MB

func main() {
	defer handleExit()

	resources, teardown := config.Load(os.Stdout)
	defer teardown()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	mcpServer, err := newMCPServer(resources)
	if err != nil {
		resources.Logger().Error("failed to create MCP server",
			slog.String("error", err.Error()),
		)
		exit(exitCodeSetupFailure)
	}
	mcpHTTPServer := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})

	mux := newRouter(resources)
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

func newMCPServer(resources config.Resources) (*mcp.Server, error) {
	projectsGroup := twprojects.DefaultToolsetGroup(false, false, resources.TeamworkEngine())
	if err := projectsGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		return nil, fmt.Errorf("failed to enable toolsets: %w", err)
	}

	deskGroup := twdesk.DefaultToolsetGroup(resources.TeamworkHTTPClient())
	if err := deskGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		return nil, fmt.Errorf("failed to enable desk toolsets: %w", err)
	}

	return config.NewMCPServer(resources, projectsGroup, deskGroup), nil
}

func newRouter(resources config.Resources) *http.ServeMux {
	mux := http.NewServeMux()
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
  "scopes_supported": [ "projects", "desk" ]
}`))
	})
	return mux
}

func addRouterMiddlewares(resources config.Resources, mux *http.ServeMux) http.Handler {
	return limitBodyMiddleware(
		requestInfoMiddleware(
			logMiddleware(resources.Logger(),
				sentryMiddleware(
					resources,
					tracerMiddleware(
						resources,
						authMiddleware(
							resources,
							mux,
						),
					),
				),
			),
		),
	)
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var reqBody []byte
		if r.Body != nil {
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				logger.Error("failed to read request body", slog.String("error", err.Error()))
			}
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		rw := request.NewResponseWriter(w)
		next.ServeHTTP(rw, r)
		duration := time.Since(start)

		headers := r.Header.Clone()
		if auth := headers.Get("Authorization"); auth != "" {
			if authParts := strings.SplitN(auth, " ", 2); len(authParts) == 2 {
				headers.Set("Authorization", authParts[0]+" REDACTED")
			} else {
				headers.Set("Authorization", "REDACTED")
			}
		}

		var traceID string
		if info, ok := request.InfoFromContext(r.Context()); ok {
			traceID = info.TraceID
		}

		logger.Info("request",
			slog.String("trace_id", traceID),
			slog.String("request_url", r.URL.String()),
			slog.String("request_method", r.Method),
			slog.Any("request_headers", headers),
			slog.String("request_body", string(reqBody)),
			slog.Int("response_status", rw.StatusCode()),
			slog.Any("response_headers", rw.Header()),
			slog.String("response_body", string(rw.Body())),
			slog.String("duration", duration.String()),
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
	return ddhttp.WrapHandler(next, resources.Info.DatadogAPM.Service, "http.request",
		ddhttp.WithResourceNamer(func(req *http.Request) string {
			return fmt.Sprintf("%s_%s", req.Method, req.URL.Path)
		}),
		ddhttp.WithIgnoreRequest(func(req *http.Request) bool {
			if req.URL.Path == "/api/health" {
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
	}

	whitelistPrefixEndpoints := map[string][]string{
		// OAuth2 endpoints cannot require authentication
		"/.well-known": {"GET", "OPTIONS"},
	}

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

		info, err := auth.GetBearerInfo(r.Context(), resources, bearerToken)
		if err == auth.ErrBearerInfoUnauthorized {
			// https://datatracker.ietf.org/doc/html/rfc9728#name-www-authenticate-response
			w.Header().Set("WWW-Authenticate",
				`Bearer resource_metadata="`+resources.Info.MCPURL+`/.well-known/oauth-protected-resource"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		} else if err != nil {
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

		ctx := r.Context()
		// detect cross-region requests
		ctx = config.WithCrossRegion(ctx, !strings.EqualFold(resources.Info.AWSRegion, info.Region))
		// inject customer URL
		ctx = config.WithCustomerURL(ctx, info.URL)
		// inject bearer token
		ctx = config.WithBearerToken(ctx, bearerToken)
		// inject scopes
		ctx = config.WithScopes(ctx, info.Meta.Scopes)
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
