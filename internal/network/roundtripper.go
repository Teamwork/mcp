package network

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"
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
func (lrt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	var reqBody []byte
	if req.Body != nil {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			lrt.Log.Error("failed to read request body", "error", err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}

	headers := req.Header.Clone()
	if headers.Get("Authorization") != "" {
		headers.Set("Authorization", "REDACTED")
	}

	// Log the request
	lrt.Log.Info("HTTP request",
		"url", req.URL.String(),
		"method", req.Method,
		"headers", headers,
		"body", string(reqBody),
	)

	transport := lrt.Base
	if transport == nil {
		transport = http.DefaultTransport
	}

	resp, err := transport.RoundTrip(req)
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

	lrt.Log.Info("HTTP response",
		"url", req.URL.String(),
		"method", req.Method,
		"status", resp.Status,
		"headers", resp.Header,
		"body", string(respBody),
		"duration", time.Since(start).String(),
	)

	return resp, nil
}
