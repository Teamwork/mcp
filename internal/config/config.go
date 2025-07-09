package config

import (
	"log/slog"
	"net/http"
	"os"

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

	resources.logger = slog.New(logHandler)
	resources.teamworkHTTPClient = new(http.Client)
	resources.teamworkEngine = twapi.NewEngine(session.NewBearerTokenContext(),
		twapi.WithHTTPClient(resources.teamworkHTTPClient),
		twapi.WithLogger(resources.logger),
	)
	return resources
}
