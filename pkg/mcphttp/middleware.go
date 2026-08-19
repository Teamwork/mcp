package mcphttp

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/logsafe"
	"github.com/teamwork/mcp/pkg/request"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// DefaultMaxBodySize is the request body limit LimitBody applies unless a server
// passes its own. Pin the same value into
// mcp.StreamableHTTPOptions.MaxRequestBodyBytes: left at zero the SDK applies
// its own smaller default, silently tightening the limit this one advertises.
const DefaultMaxBodySize = 10 * 1024 * 1024 // 10 MB

// Chain applies middlewares so the first argument is the outermost wrapper (runs
// first on the request, last on the response).
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// StripProfile checks whether the request path starts with a known profile name,
// and if so strips it and sets a "TW-MCP-Profile" header. This lets clients use
// URLs like "/project-manager/endpoint" to reach "/endpoint" with a profile
// context.
func StripProfile(profiles []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the path starts with a known profile, strip it and set a header
		r.Header.Set("TW-MCP-Profile", "")
		for _, profile := range profiles {
			if strings.HasPrefix(r.URL.Path, "/"+profile+"/") {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, "/"+profile)
				r.Header.Add("TW-MCP-Profile", profile)
				break
			}
		}
		next.ServeHTTP(w, r)
	})
}

// LimitBody caps the request body a client may send.
func LimitBody(maxBodySize int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		next.ServeHTTP(w, r)
	})
}

// RequestInfo attaches the per-request trace and installation info every later
// middleware and the MCP logging middleware read.
func RequestInfo(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(request.WithInfo(r.Context(), request.NewInfo(r))))
	})
}

// Log records one line per request: the trace id, the request and response, and
// the installation and user the token resolved to. Request bodies go through
// logsafe, which strips anything token-shaped.
//
// skipPaths keeps the noisy endpoints out: health checks fire constantly, and an
// SSE stream lives as long as the request, so its body is logged by SSELog
// instead.
func Log(logger *slog.Logger, skipPaths map[string]struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, skip := skipPaths[r.URL.Path]; skip {
			next.ServeHTTP(w, r)
			return
		}

		var reqBody []byte
		if r.Body != nil {
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				logger.Error("failed to read request body", slog.String("error", err.Error()))
			}
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		start := time.Now()
		rw := request.NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		headers := r.Header.Clone()
		if auth := headers.Get("Authorization"); auth != "" {
			if authParts := strings.SplitN(auth, " ", 2); len(authParts) == 2 {
				headers.Set("Authorization", authParts[0]+" REDACTED")
			} else {
				headers.Set("Authorization", "REDACTED")
			}
		}

		info, _ := request.InfoFromContext(r.Context())
		logger.Info("request",
			slog.String("trace_id", info.TraceID()),
			slog.String("request_url", r.URL.String()),
			slog.String("request_method", r.Method),
			slog.Any("request_headers", headers),
			slog.String("request_body", logsafe.String(string(reqBody))),
			slog.Int("response_status", rw.StatusCode()),
			slog.Any("response_headers", rw.Header()),
			slog.String("response_body", string(rw.Body())),
			slog.Duration("duration", time.Since(start)),
			slog.Int64("installation.id", info.InstallationID()),
			slog.String("installation.url", info.InstallationURL()),
			slog.Int64("user.id", info.UserID()),
		)
	})
}

// SSELog logs a Server-Sent Events endpoint, where the long-lived GET stream and
// the short-lived POST message deliveries need different treatment: the stream
// is logged once when it opens and once when it closes, so a connection held for
// hours does not sit unlogged, or buffer its body until it ends.
func SSELog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := r.Header.Clone()
		if auth := headers.Get("Authorization"); auth != "" {
			if authParts := strings.SplitN(auth, " ", 2); len(authParts) == 2 {
				headers.Set("Authorization", authParts[0]+" REDACTED")
			} else {
				headers.Set("Authorization", "REDACTED")
			}
		}

		info, _ := request.InfoFromContext(r.Context())

		if r.Method == http.MethodGet {
			// long-lived SSE stream connection
			logger.Info("SSE stream opened",
				slog.String("trace_id", info.TraceID()),
				slog.String("request_url", r.URL.String()),
				slog.String("request_method", r.Method),
				slog.Any("request_headers", headers),
				slog.String("path", r.URL.String()),
			)
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("SSE stream closed",
				slog.String("trace_id", info.TraceID()),
				slog.Duration("duration", time.Since(start)),
			)
			return
		}

		// short-lived message deliveries to a session

		sessionID := r.URL.Query().Get("sessionid")

		var reqBody []byte
		if r.Body != nil {
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				logger.Error("failed to read SSE message body", slog.String("error", err.Error()))
			}
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		start := time.Now()
		rw := request.NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		logger.Info("SSE message",
			slog.String("trace_id", info.TraceID()),
			slog.String("session_id", sessionID),
			slog.String("request_url", r.URL.String()),
			slog.String("request_method", r.Method),
			slog.Any("request_headers", headers),
			slog.String("request_body", logsafe.String(string(reqBody))),
			slog.Int("response_status", rw.StatusCode()),
			slog.Any("response_headers", rw.Header()),
			slog.String("response_body", string(rw.Body())),
			slog.Duration("duration", time.Since(start)),
			slog.Int64("installation.id", info.InstallationID()),
			slog.String("installation.url", info.InstallationURL()),
			slog.Int64("user.id", info.UserID()),
		)
	})
}

// Sentry scopes a Sentry hub to the request so a panic or reported error carries
// the request that caused it. A no-op when no DSN is configured.
func Sentry(resources config.Resources, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if resources.Info.Log.SentryDSN != "" {
			hub := sentry.CurrentHub().Clone()
			hub.Scope().SetRequest(r)
			ctx := sentry.SetHubOnContext(r.Context(), hub)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// Tracer wraps the handler in OpenTelemetry tracing, naming each span by method
// and path. Returns next unchanged when tracing is disabled.
//
// skipPaths drops the endpoints not worth a trace: health checks are constant
// noise, and a long-lived SSE stream does not fit a request span. Every
// /.well-known path is skipped too.
func Tracer(resources config.Resources, skipPaths map[string]struct{}, next http.Handler) http.Handler {
	if !resources.Info.OTel.Enabled {
		return next
	}
	return otelhttp.NewHandler(next, "http.request",
		otelhttp.WithSpanNameFormatter(func(_ string, req *http.Request) string {
			return fmt.Sprintf("%s_%s", req.Method, req.URL.Path)
		}),
		// WithFilter keeps what it returns true for, the inverse of the paths above.
		otelhttp.WithFilter(func(req *http.Request) bool {
			if _, skip := skipPaths[req.URL.Path]; skip {
				return false
			}
			return !strings.HasPrefix(req.URL.Path, "/.well-known")
		}),
	)
}
