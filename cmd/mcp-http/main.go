package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/cli"
	"github.com/teamwork/mcp/internal/twchat"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
	"github.com/teamwork/mcp/pkg/auth"
	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/mcphttp"
	"github.com/teamwork/mcp/pkg/toolsets"
)

var methods = cli.NewMethods(toolsets.MethodAll)

// Limit request body size (e.g., 10MB)
const maxBodySize = mcphttp.DefaultMaxBodySize

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

	mux := newRouter(resources, groups)
	mux.Handle("/sse", mcphttp.SSELog(resources.Logger(), mcpSSEServer))
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

func newRouter(resources config.Resources, groups []*toolsets.ToolsetGroup) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/favicon.ico", http.RedirectHandler("https://teamwork.com/favicon.ico", http.StatusPermanentRedirect))
	mcphttp.Health(mux, "/api/health")
	mcphttp.ProtectedResource(mux, resources, groups)
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

// quietPaths are the endpoints not worth a log line or a trace: health checks
// fire constantly, browsers probe the favicon, and /sse is logged by
// mcphttp.SSELog instead, which handles its long-lived stream.
var quietPaths = map[string]struct{}{
	"/favicon.ico": {},
	"/api/health":  {},
	"/sse":         {},
}

func addRouterMiddlewares(resources config.Resources, mux *http.ServeMux) http.Handler {
	validator := auth.NewValidator(resources.TeamworkHTTPClient(), resources.Info.APIURL, resources.Logger())

	return mcphttp.Chain(mux,
		func(h http.Handler) http.Handler { return mcphttp.StripProfile(resources.Info.MCPProfiles, h) },
		htmlIndexMiddleware,
		func(h http.Handler) http.Handler { return mcphttp.LimitBody(maxBodySize, h) },
		mcphttp.RequestInfo,
		func(h http.Handler) http.Handler { return mcphttp.Log(resources.Logger(), quietPaths, h) },
		func(h http.Handler) http.Handler { return mcphttp.Sentry(resources, h) },
		func(h http.Handler) http.Handler { return mcphttp.Tracer(resources, quietPaths, h) },
		func(h http.Handler) http.Handler { return mcphttp.Auth(resources, validator, h) },
	)
}

// htmlIndexMiddleware redirects a browser hitting the root to the MCP homepage.
// Specific to this server: a premium or internal deployment has no such page.
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
