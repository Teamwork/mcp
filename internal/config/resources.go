package config

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

// Resources stores all the resources loaded in the startup.
type Resources struct {
	teamworkHTTPClient *http.Client
	logger             *slog.Logger

	// Info stores environment variables mappings.
	Info struct {
		// ServerAddress is the address of the server.
		ServerAddress string
		// Environment is the environment this app is running in.
		Environment string
		// DevEnvInstallation is the Teamwork DevEnv installation URL.
		DevEnvInstallation string
	}
}

func load(resources *Resources) {
	resources.Info.ServerAddress = viper.GetString("server.address")
	resources.Info.Environment = viper.GetString("env")
	resources.Info.DevEnvInstallation = viper.GetString("devenv.installation")
}

// Logger returns the logger resource.
func (r *Resources) Logger() *slog.Logger {
	return r.logger
}

// IsDev returns true if the app is running in development environment.
func (r *Resources) IsDev() bool {
	return strings.EqualFold(r.Info.Environment, "dev")
}

// TeamworkHTTPClient returns the HTTP client to be used to make requests to
// Teamwork API.
func (r *Resources) TeamworkHTTPClient() *http.Client {
	return r.teamworkHTTPClient
}
