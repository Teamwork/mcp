package config

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load loads the configuration for the MCP service.
func Load() Resources {
	var resources Resources
	configureViper(&resources)

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

// configureViper configures the viper instance.
func configureViper(resources *Resources) {
	viper := viper.GetViper()
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	defaults(viper)
	load(resources)
}
