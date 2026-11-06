package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/joekings2k/staypass-be/internal/platform/config"
)

func NewLogger(cfg *config.Config) *slog.Logger {
	level := slog.LevelDebug
	if cfg.IsProduction() {
		level = slog.LevelInfo
	}

	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

type ctxKey int

const loggerKey ctxKey = iota

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
