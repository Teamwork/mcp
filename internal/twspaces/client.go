package twspaces

import (
	"context"
	"log/slog"
	"net/http"

	spacesclient "github.com/teamwork/spacessdkgo/client"
	"github.com/teamwork/mcp/internal/config"
)

// ClientFromContext creates a new Spaces client with the correct base URL based
// on the context. It uses the customer URL from context if available, otherwise
// defaults to https://www.teamwork.com. It also extracts the bearer token from
// the context and passes it via WithAPIKey. The spaces SDK normalizes the URL
// to include /spaces/api/v1 automatically.
func ClientFromContext(ctx context.Context, httpClient *http.Client) *spacesclient.Client {
	baseURL := "https://www.teamwork.com"

	// Override with customer URL if present in context
	if customerURL, ok := config.CustomerURLFromContext(ctx); ok {
		baseURL = customerURL
	}

	options := []spacesclient.Option{
		spacesclient.WithHTTPClient(httpClient),
	}

	// Pass the bearer token from context if available
	if bearerToken, ok := config.BearerTokenFromContext(ctx); ok {
		options = append(options, spacesclient.WithAPIKey(bearerToken))
	}

	// Pass the logger from context if available
	if logger := slog.Default(); logger != nil {
		options = append(options, spacesclient.WithLogger(logger))
	}

	return spacesclient.NewClient(baseURL, options...)
}
