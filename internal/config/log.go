package config

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/getsentry/sentry-go"
	sentryslog "github.com/getsentry/sentry-go/slog"
)

// customLogHandler is a slog.Handler that wraps another slog.Handler and
// sends error-level logs to Sentry if a Sentry DSN is configured.
type customLogHandler struct {
	handler slog.Handler
	sentry  slog.Handler
}

// Enabled reports whether the handler handles records at the given level.
func (h *customLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle processes a log record. If the log level is error or higher and
// Sentry is configured, it also sends the log to Sentry.
func (h *customLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.sentry != nil && h.sentry.Enabled(ctx, record.Level) {
		if err := h.sentry.Handle(ctx, record); err != nil {
			return err
		}
	}
	return h.handler.Handle(ctx, record)
}

// WithAttrs returns a new handler that includes the given attributes.
func (h *customLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	logHandler := &customLogHandler{
		handler: h.handler.WithAttrs(attrs),
		sentry:  h.sentry,
	}
	if logHandler.sentry != nil {
		logHandler.sentry = logHandler.sentry.WithAttrs(attrs)
	}
	return logHandler
}

// WithGroup returns a new handler that includes the given group.
func (h *customLogHandler) WithGroup(name string) slog.Handler {
	logHandler := &customLogHandler{
		handler: h.handler.WithGroup(name),
		sentry:  h.sentry,
	}
	if logHandler.sentry != nil {
		logHandler.sentry = logHandler.sentry.WithGroup(name)
	}
	return logHandler
}

func newCustomLogHandler(resources Resources) slog.Handler {
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(resources.Info.Log.Level)); err != nil {
		logLevel = slog.LevelInfo
	}

	var handler, sentryHandler slog.Handler
	if strings.EqualFold(resources.Info.Log.Format, "json") {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	}

	if resources.Info.Log.SentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:            resources.Info.Log.SentryDSN,
			EnableTracing:  true,
			SendDefaultPII: true,
			Release:        resources.Info.Version,
			Environment:    resources.Info.Environment,
		})
		if err != nil {
			slog.Default().Error("failed to initialize sentry",
				slog.String("error", err.Error()),
			)
		} else {
			sentryHandler = sentryslog.Option{
				EventLevel: []slog.Level{
					slog.LevelError,
					sentryslog.LevelFatal,
				},
			}.NewSentryHandler(context.Background())
		}
	}

	return &customLogHandler{
		handler: handler,
		sentry:  sentryHandler,
	}
}
