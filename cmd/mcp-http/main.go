package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	ddhttp "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/getsentry/sentry-go"
	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/auth"
	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/request"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk/session"
)

var reBearerToken = regexp.MustCompile(`^Bearer (.+)$`)

func main() {
	defer handleExit()

	resources, teardown := config.Load()
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
	mcpHTTPServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/"),
		server.WithStateLess(true),
	)

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

func newMCPServer(resources config.Resources) (*server.MCPServer, error) {
	group := twprojects.DefaultToolsetGroup(false, false, resources.TeamworkEngine())
	if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
		return nil, fmt.Errorf("failed to enable toolsets: %w", err)
	}
	return config.NewMCPServer(resources, group), nil
}

func newRouter(resources config.Resources) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodOptions {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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

		_, _ = w.Write([]byte(`{
  "resource": "` + resources.Info.MCPURL + `",
  "authorization_servers": ["` + resources.Info.APIURL + `"],
  "bearer_methods_supported": ["header"],
  "resource_documentation": "https://apidocs.teamwork.com/guides/teamwork/app-login-flow"
}`))
	})
	return mux
}

func addRouterMiddlewares(resources config.Resources, mux *http.ServeMux) http.Handler {
	return sentryMiddleware(resources, requestInfoMiddleware(tracerMiddleware(resources, authMiddleware(resources, mux))))
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

func requestInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(request.WithInfo(r.Context(), request.NewInfo(r))))
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// some endpoints don't require auth
		if (r.URL.Path == "/api/health" || strings.HasPrefix(r.URL.Path, "/.well-known")) &&
			(r.Method == http.MethodGet || r.Method == http.MethodOptions) {
			next.ServeHTTP(w, r)
			return
		}

		requestLogger := resources.Logger().With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
		)

		matches := reBearerToken.FindStringSubmatch(r.Header.Get("Authorization"))
		if len(matches) < 2 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		bearerToken := matches[1]

		info, err := auth.GetBearerInfo(r.Context(), resources, bearerToken)
		if err == auth.ErrBearerInfoUnauthorized {
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
