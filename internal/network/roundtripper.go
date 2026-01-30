package network

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/teamwork/mcp/internal/request"
)

// LoggingRoundTripper is an http.RoundTripper that logs requests and responses
type LoggingRoundTripper struct {
	Base http.RoundTripper
	Log  *slog.Logger
}

// NewLoggingRoundTripper creates a new LoggingRoundTripper with the given logger
func NewLoggingRoundTripper(logger *slog.Logger, base http.RoundTripper) *LoggingRoundTripper {
	return &LoggingRoundTripper{
		Log:  logger,
		Base: base,
	}
}

// RoundTrip implements the RoundTripper interface
func (lrt *LoggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()

	var reqBody []byte
	if r.Body != nil {
		var err error
		reqBody, err = io.ReadAll(r.Body)
		if err != nil {
			lrt.Log.Error("failed to read request body", slog.String("error", err.Error()))
		}
		r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}

	headers := r.Header.Clone()
	if auth := headers.Get("Authorization"); auth != "" {
		if authParts := strings.SplitN(auth, " ", 2); len(authParts) == 2 {
			headers.Set("Authorization", authParts[0]+" REDACTED")
		} else {
			headers.Set("Authorization", "REDACTED")
		}
	}

	var traceID string
	if info, ok := request.InfoFromContext(r.Context()); ok {
		traceID = info.TraceID
	}

	transport := lrt.Base
	if transport == nil {
		transport = http.DefaultTransport
	}

	resp, err := transport.RoundTrip(r)
	if err != nil {
		lrt.Log.Error("HTTP request failed", "error", err)
		return resp, err
	}

	var respBody []byte
	if resp.Body != nil {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			lrt.Log.Error("failed to read response body", "error", err)
		}

		resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	}

	lrt.Log.Info("internal request",
		slog.String("trace_id", traceID),
		slog.String("request_url", r.URL.String()),
		slog.String("request_method", r.Method),
		slog.Any("request_headers", headers),
		slog.String("request_body", string(reqBody)),
		slog.Int("response_status", resp.StatusCode),
		slog.Any("response_headers", resp.Header),
		slog.String("response_body", string(respBody)),
		slog.String("duration", time.Since(start).String()),
	)

	return resp, nil
}
