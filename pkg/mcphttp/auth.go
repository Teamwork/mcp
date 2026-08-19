// Package mcphttp carries the HTTP plumbing an MCP server needs around the
// protocol handler: the middleware chain, the health and RFC 9728 metadata
// endpoints, and the bearer-token authentication that turns a token into the
// per-request values tool handlers read.
//
// It lives here rather than in a server's main package because a main package
// cannot be imported. Authentication especially must not be forked: two copies
// drift, and the one that drifts is the one that stops rejecting what it should.
package mcphttp

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/teamwork/mcp/pkg/auth"
	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/request"
	"github.com/teamwork/mcp/pkg/twctx"
	"github.com/teamwork/twapi-go-sdk/session"
)

var reBearerToken = regexp.MustCompile(`^Bearer (.+)$`)

// Auth authenticates every request with a bearer token, rejecting the ones that
// carry no usable credential and populating the context of the ones that do.
//
// Unauthenticated paths are the ones that cannot require a token: health checks,
// browser favicon probes, and the /.well-known OAuth metadata an unauthorised
// client fetches to discover where to authorise. A request with no
// Authorization header on any other path is allowed through only when its JSON
// body names a protocol method auth.Bypass whitelists, which is how a client
// negotiates capabilities before it holds a token.
func Auth(resources config.Resources, validator *auth.Validator, next http.Handler) http.Handler {
	whitelistEndpoints := map[string][]string{
		// health checks don't require authentication
		"/api/health": {http.MethodGet, http.MethodOptions},
		// browser may request favicons without authentication
		"/favicon.ico": {http.MethodGet, http.MethodOptions},
	}

	whitelistPrefixEndpoints := map[string][]string{
		// OAuth2 endpoints cannot require authentication
		"/.well-known": {http.MethodGet, http.MethodOptions},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestLogger := resources.Logger().With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
		)

		if r.Header.Get("Authorization") == "" {
			// some endpoints don't require auth

			if methods, ok := whitelistEndpoints[r.URL.Path]; ok && slices.Contains(methods, r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			for prefix, methods := range whitelistPrefixEndpoints {
				if strings.HasPrefix(r.URL.Path, prefix) && slices.Contains(methods, r.Method) {
					next.ServeHTTP(w, r)
					return
				}
			}

			content, err := io.ReadAll(r.Body)
			if err != nil {
				requestLogger.ErrorContext(r.Context(), "failed to read request body",
					slog.String("error", err.Error()),
				)
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}

			bypass, err := auth.Bypass(content)
			switch {
			case err != nil, !bypass:
				challenge(w, resources)
			default:
				r.Body = io.NopCloser(bytes.NewBuffer(content))
				next.ServeHTTP(w, r)
			}
			return
		}

		matches := reBearerToken.FindStringSubmatch(r.Header.Get("Authorization"))
		if len(matches) < 2 {
			challenge(w, resources)
			return
		}
		bearerToken := matches[1]

		info, err := validator.GetBearerInfo(r.Context(), bearerToken)
		switch {
		case errors.Is(err, auth.ErrBearerInfoUnauthorized):
			// The token was positively rejected, so challenge the client to
			// re-authorise.
			challenge(w, resources)
			return

		case errors.Is(err, auth.ErrBearerInfoCanceled):
			// The client hung up while we were validating. Nobody is left to
			// read a response and nothing failed on our side, so this is debug
			// noise rather than an error.
			requestLogger.DebugContext(r.Context(), "bearer info validation canceled by client",
				slog.String("error", err.Error()),
			)
			return

		case errors.Is(err, auth.ErrBearerInfoUnavailable):
			// We could not determine whether the token is valid. Never send a
			// re-authorisation challenge here: the token is probably fine, and
			// telling the client to discard it puts the user through the whole
			// OAuth flow (and MFA) over what may be a momentary blip.
			requestLogger.ErrorContext(r.Context(), "failed to validate bearer info",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to validate bearer token", http.StatusServiceUnavailable)
			return

		case err != nil:
			requestLogger.ErrorContext(r.Context(), "failed to get bearer info",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Failed to get bearer info", http.StatusInternalServerError)
			return
		}

		if span, ok := tracer.SpanFromContext(r.Context()); ok {
			span.SetTag("user.id", info.UserID)
			span.SetTag("installation.id", info.InstallationID)
			span.SetTag("installation.url", info.URL)
		}
		if requestInfo, ok := request.InfoFromContext(r.Context()); ok {
			requestInfo.SetAuth(info.InstallationID, info.URL, info.UserID)
		}

		ctx := r.Context()
		// detect cross-region requests
		ctx = twctx.WithCrossRegion(ctx, !strings.EqualFold(resources.Info.AWSRegion, info.Region))
		// inject customer URL
		ctx = twctx.WithCustomerURL(ctx, info.URL)
		// inject bearer token
		ctx = twctx.WithBearerToken(ctx, bearerToken)
		// inject scopes
		ctx = twctx.WithScopes(ctx, info.Meta.Scopes)
		// inject session
		ctx = session.WithBearerTokenContext(ctx, session.NewBearerToken(bearerToken, info.URL))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// challenge answers 401 with the RFC 9728 pointer to this server's
// protected-resource metadata, which is how a client learns where to authorise.
// Only send it when the token was actually refused — it makes the client throw
// its token away.
//
// https://datatracker.ietf.org/doc/html/rfc9728#name-www-authenticate-response
func challenge(w http.ResponseWriter, resources config.Resources) {
	w.Header().Set("WWW-Authenticate",
		`Bearer resource_metadata="`+resources.Info.MCPURL+`/.well-known/oauth-protected-resource"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
