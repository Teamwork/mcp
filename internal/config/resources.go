package config

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	twapi "github.com/teamwork/twapi-go-sdk"
)

// Resources stores all the resources loaded in the startup.
type Resources struct {
	teamworkHTTPClient *http.Client
	teamworkEngine     *twapi.Engine
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

func newResources() Resources {
	var resources Resources
	resources.Info.ServerAddress = getEnv("SERVER_ADDRESS", "localhost:8012")
	resources.Info.Environment = getEnv("ENV", "dev")
	resources.Info.DevEnvInstallation = getEnv("DEVENV_INSTALLATION", "")
	return resources
}

// Logger returns the logger resource.
func (r *Resources) Logger() *slog.Logger {
	return r.logger
}

// IsDev returns true if the app is running in development environment.
func (r *Resources) IsDev() bool {
	return strings.EqualFold(r.Info.Environment, "dev")
}

// IsStaging returns true if the app is running in staging environment.
func (r *Resources) IsStaging() bool {
	return strings.EqualFold(r.Info.Environment, "staging")
}

// TeamworkHTTPClient returns the HTTP client to be used to make requests to
// Teamwork API.
func (r *Resources) TeamworkHTTPClient() *http.Client {
	return r.teamworkHTTPClient
}

// TeamworkEngine returns the Teamwork Engine instance to be used to make
// requests to Teamwork API.
func (r *Resources) TeamworkEngine() *twapi.Engine {
	return r.teamworkEngine
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
