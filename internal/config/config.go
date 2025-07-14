package config

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/teamwork/mcp/internal/request"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/session"
)

// Load loads the configuration for the MCP service.
func Load() Resources {
	resources := newResources()

	var logHandler slog.Handler
	if resources.IsDev() {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	var haProxyURL *url.URL
	if resources.Info.HAProxyURL != "" {
		var err error
		if haProxyURL, err = url.Parse(resources.Info.HAProxyURL); err != nil {
			resources.logger.Error("failed to parse HAProxy URL",
				slog.String("url", resources.Info.HAProxyURL),
				slog.String("error", err.Error()),
			)
			// reset to nil to avoid using an invalid URL
			haProxyURL = nil
		}
	}

	resources.logger = slog.New(logHandler)
	resources.teamworkHTTPClient = new(http.Client)
	resources.teamworkEngine = twapi.NewEngine(session.NewBearerTokenContext(),
		twapi.WithHTTPClient(resources.teamworkHTTPClient),
		twapi.WithMiddleware(func(next twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
				request.SetProxyHeaders(req)
				return next.Do(req)
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
	return resources
}
