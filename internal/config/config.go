package config

import (
	"log/slog"
	"net/http"
	"os"
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
	return resources
}
