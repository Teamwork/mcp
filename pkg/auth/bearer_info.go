package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// ErrBearerInfoUnauthorized is returned when Teamwork API positively rejected
// the bearer token. This is the only error that may be turned into a 401 for
// the MCP client: a 401 carries an RFC 9728 re-authorisation challenge, which
// makes the client discard the token and run the OAuth flow again.
var ErrBearerInfoUnauthorized = errors.New("unauthorized: failed to get bearer info")

// ErrBearerInfoUnavailable is returned when the token could not be validated at
// all — Teamwork API was unreachable, timed out, or answered with a server
// error. The token may well be valid, so callers must not tell the client to
// throw it away; report this as 503 instead.
var ErrBearerInfoUnavailable = errors.New("unavailable: failed to validate bearer info")

// ErrBearerInfoCanceled is returned when the caller's own request context was
// cancelled while validating, which normally means the MCP client hung up
// first. Nothing went wrong on our side and no response will be read, so this
// is not worth reporting as a server error.
var ErrBearerInfoCanceled = errors.New("canceled: bearer info validation abandoned")

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

// Validator resolves a bearer token into the installation and user it belongs
// to. It takes the HTTP client, API base URL and logger explicitly rather than
// the server configuration, so a server built on this package can supply its
// own.
type Validator struct {
	client *http.Client
	apiURL string
	logger *slog.Logger
}

// NewValidator creates a Validator that asks apiURL to identify bearer tokens.
func NewValidator(client *http.Client, apiURL string, logger *slog.Logger) *Validator {
	return &Validator{client: client, apiURL: apiURL, logger: logger}
}

// GetBearerInfo retrieves information about the bearer token from Teamwork API.
// It returns a BearerInfo struct containing the user ID, installation ID, and
// installation URL. If the token is invalid or unauthorized, it returns
// ErrBearerInfoUnauthorized.
func (v *Validator) GetBearerInfo(ctx context.Context, token string) (*BearerInfo, error) {
	url := v.apiURL + "/launchpad/v1/userinfo.json"
	authRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}
	authRequest.Header.Set("Authorization", "Bearer "+token)

	response, err := v.client.Do(authRequest)
	if err != nil {
		// Distinguish "the client went away" from "we could not reach Teamwork
		// API": the former is routine (an MCP client closing its stream) and
		// must not be logged as a server error, the latter is worth alerting on.
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("%w: %w", ErrBearerInfoCanceled, err)
		}
		return nil, fmt.Errorf("%w: failed to perform auth request: %w", ErrBearerInfoUnavailable, err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			v.logger.ErrorContext(ctx, "failed to close auth response body",
				slog.String("error", err.Error()),
			)
		}
	}()

	if err := classifyAuthStatus(response.StatusCode, url); err != nil {
		return nil, err
	}

	var info BearerInfo

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&info); err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("%w: %w", ErrBearerInfoCanceled, err)
		}
		return nil, fmt.Errorf("%w: failed to decode auth response: %w", ErrBearerInfoUnavailable, err)
	}
	return &info, nil
}

// classifyAuthStatus turns the status code of a userinfo response into an
// error, or nil when the token was validated.
//
// Only 401 and 403 mean the token itself was refused. Every other non-200 —
// 5xx, 429, a proxy error page — means validation never happened, and must stay
// distinguishable: reporting those as a rejection is what let a momentary
// upstream failure look to the OAuth client like an expired token, costing the
// user a full re-authorisation.
func classifyAuthStatus(statusCode int, url string) error {
	switch statusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrBearerInfoUnauthorized
	default:
		return fmt.Errorf("%w: unexpected status %d from %s", ErrBearerInfoUnavailable, statusCode, url)
	}
}
