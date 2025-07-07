package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/config"
)

const (
	mcpName    = "Teamwork.com"
	mcpVersion = "1.0.0"
)

func main() {
	resources := config.Load()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	mcpServer := server.NewMCPServer(mcpName, mcpVersion,
		server.WithRecovery(),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	mux := http.NewServeMux()
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
		if resources.IsDev() && resources.Info.DevEnvInstallation != "" {
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

	mcpHTTPServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/"),
		server.WithStateLess(true),
	)
	mux.Handle("/", mcpHTTPServer)

	httpServer := &http.Server{
		Addr:    resources.Info.ServerAddress,
		Handler: authMiddleware(resources, mux),
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

func authMiddleware(resources config.Resources, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// some endpoints don't require auth
		if strings.HasPrefix(r.URL.Path, "/.well-known") && (r.Method == http.MethodGet || r.Method == http.MethodOptions) {
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

		authRequest, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			requestLogger.Error("failed to create auth request",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to create auth request", http.StatusInternalServerError)
			return
		}
		authRequest.Header.Set("Authorization", r.Header.Get("Authorization"))

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

		requestLogger.Info("authenticated request",
			slog.Int64("user_id", info.UserID),
			slog.Int64("installation_id", info.InstallationID),
			slog.String("url", info.URL),
		)

		// TODO: inject auth info into context for use in tools

		next.ServeHTTP(w, r)
	})
}

type authInfo struct {
	UserID         int64  `json:"user_id"`
	InstallationID int64  `json:"installation_id"`
	URL            string `json:"url"`
}
