package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/request"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk/session"
)

const (
	mcpName    = "Teamwork.com"
	mcpVersion = "1.0.0"
)

var reBearerToken = regexp.MustCompile(`^Bearer (.+)$`)

func main() {
	defer handleExit()
	resources := config.Load()

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
	mcpServer := server.NewMCPServer(mcpName, mcpVersion,
		server.WithRecovery(),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	group := twprojects.DefaultToolsetGroup(false, resources.TeamworkEngine())
	if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
		return nil, fmt.Errorf("failed to enable toolsets: %w", err)
	}
	group.RegisterAll(mcpServer)

	return mcpServer, nil
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

		resourceAddress := "https://mcp.teamwork.com"
		authorizationAddress := "https://www.teamwork.com"

		switch {
		case resources.IsStaging():
			resourceAddress = "https://mcp.eks.stg.teamworkops.com"
			authorizationAddress = "https://www.staging.teamwork.com"
		case resources.IsDev() && resources.Info.DevEnvInstallation != "":
			resourceAddress = fmt.Sprintf("https://mcp.%s", resources.Info.DevEnvInstallation)
			authorizationAddress = fmt.Sprintf("https://%s", resources.Info.DevEnvInstallation)
		}

		_, _ = w.Write([]byte(`{
  "resource": "` + resourceAddress + `",
  "authorization_servers": ["` + authorizationAddress + `"],
  "bearer_methods_supported": ["header"],
  "resource_documentation": "https://apidocs.teamwork.com/guides/teamwork/app-login-flow"
}`))
	})
	return mux
}

func addRouterMiddlewares(resources config.Resources, mux *http.ServeMux) http.Handler {
	return requestInfoMiddleware(tracerMiddleware(resources, authMiddleware(resources, mux)))
}

func requestInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(request.WithInfo(r.Context(), request.NewInfo(r))))
	})
}

func tracerMiddleware(resources config.Resources, next http.Handler) http.Handler {
	return httptrace.WrapHandler(next, resources.Info.DatadogAPMService, "http.request",
		httptrace.WithResourceNamer(func(req *http.Request) string {
			return fmt.Sprintf("%s_%s", req.Method, req.URL.Path)
		}),
		httptrace.WithIgnoreRequest(func(req *http.Request) bool {
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

		server := "https://www.teamwork.com"
		if resources.IsDev() && resources.Info.DevEnvInstallation != "" {
			server = fmt.Sprintf("https://%s", resources.Info.DevEnvInstallation)
		}
		url := fmt.Sprintf("%s/launchpad/v1/userinfo.json", server)

		matches := reBearerToken.FindStringSubmatch(r.Header.Get("Authorization"))
		if len(matches) < 2 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		bearerToken := matches[1]

		authRequest, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			requestLogger.Error("failed to create auth request",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to create auth request", http.StatusInternalServerError)
			return
		}
		authRequest.Header.Set("Authorization", "Bearer "+bearerToken)

		response, err := resources.TeamworkHTTPClient().Do(authRequest)
		if err != nil {
			requestLogger.Error("failed to perform auth request",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to perform auth request", http.StatusInternalServerError)
			return
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				requestLogger.Error("failed to close auth response body",
					slog.String("error", err.Error()),
				)
			}
		}()

		if response.StatusCode != http.StatusOK {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var info authInfo

		decoder := json.NewDecoder(response.Body)
		if err := decoder.Decode(&info); err != nil {
			requestLogger.Error("failed to decode auth response",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to decode auth response", http.StatusInternalServerError)
			return
		}

		requestLogger.Debug("authenticated request",
			slog.Int64("user_id", info.UserID),
			slog.Int64("installation_id", info.InstallationID),
			slog.String("url", info.URL),
		)
		if span, ok := tracer.SpanFromContext(r.Context()); ok {
			span.SetTag("user.id", info.UserID)
			span.SetTag("installation.id", info.InstallationID)
			span.SetTag("installation.url", info.URL)
		}

		r = r.WithContext(session.WithBearerTokenContext(r.Context(), session.NewBearerToken(bearerToken, info.URL)))

		next.ServeHTTP(w, r)
	})
}

type authInfo struct {
	UserID         int64  `json:"user_id"`
	InstallationID int64  `json:"installation_id"`
	URL            string `json:"url"`
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
