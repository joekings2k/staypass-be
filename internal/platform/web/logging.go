package web

import (
	"context"
	"log/slog"
)

type ctxKey int

const loggerKey ctxKey = iota

// WithLogger stores a request-scoped logger (already enriched with request_id,
// method, path) on the context so downstream layers can log with that context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// LoggerFrom returns the request-scoped logger, or the default logger if none
// was set (e.g. in tests or background jobs). Never returns nil.
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
