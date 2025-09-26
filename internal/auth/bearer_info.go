package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/teamwork/mcp/internal/config"
)

// ErrBearerInfoUnauthorized is returned when the bearer token is invalid or
// unauthorized.
var ErrBearerInfoUnauthorized = errors.New("unauthorized: failed to get bearer info")

// BearerInfo contains information about the bearer token used to authenticate
// with Teamwork API.
type BearerInfo struct {
	UserID         int64  `json:"user_id"`
	InstallationID int64  `json:"installation_id"`
	Region         string `json:"awsRegion"`
	URL            string `json:"url"`
	Meta           struct {
		Scopes []string `json:"scopes"`
	} `json:"meta"`
}

// GetBearerInfo retrieves information about the bearer token from Teamwork API.
// It returns a BearerInfo struct containing the user ID, installation ID, and
// installation URL. If the token is invalid or unauthorized, it returns
// BearerInfoUnauthorizedError.
func GetBearerInfo(ctx context.Context, resources config.Resources, token string) (*BearerInfo, error) {
	url := resources.Info.APIURL + "/launchpad/v1/userinfo.json"
	authRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}
	authRequest.Header.Set("Authorization", "Bearer "+token)

	response, err := resources.TeamworkHTTPClient().Do(authRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to perform auth request: %w", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			resources.Logger().ErrorContext(ctx, "failed to close auth response body",
				slog.String("error", err.Error()),
			)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return nil, ErrBearerInfoUnauthorized
	}

	var info BearerInfo

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode auth response: %w", err)
	}
	return &info, nil
}
