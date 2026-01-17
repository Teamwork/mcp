package twdesk

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/internal/config"
)

// ClientFromContext creates a new Desk client with the correct base URL based
// on the context. It uses the customer URL from context if available, otherwise
// defaults to https://api.teamwork.com/desk/api/v2 It also extracts the bearer
// token from the context and passes it via WithAPIKey.
func ClientFromContext(ctx context.Context, httpClient *http.Client) *deskclient.Client {
	baseURL := "https://api.teamwork.com/desk/api/v2"

	// Override with customer URL if present in context
	if customerURL, ok := config.CustomerURLFromContext(ctx); ok {
		customerURL = strings.TrimSuffix(customerURL, "/")
		baseURL = customerURL + "/desk/api/v2"
	}

	options := []deskclient.Option{
		deskclient.WithHTTPClient(httpClient),
	}

	// Pass the bearer token from context if available
	if bearerToken, ok := config.BearerTokenFromContext(ctx); ok {
		options = append(options, deskclient.WithAPIKey(bearerToken))
	}

	// Pass the logger from context if available
	if logger := slog.Default(); logger != nil {
		options = append(options, deskclient.WithLogger(logger))
	}

	return deskclient.NewClient(baseURL, options...)
}
