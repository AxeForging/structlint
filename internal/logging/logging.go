package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const loggerKey contextKey = "logger"

// CLIHandler is a custom handler that outputs CLI-friendly messages
type CLIHandler struct {
	level slog.Level
}

func (h *CLIHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *CLIHandler) Handle(ctx context.Context, record slog.Record) error {
	// Simple CLI-friendly format: just the message, no timestamp or level
	fmt.Fprintln(os.Stderr, record.Message)
	return nil
}

func (h *CLIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // Ignore attributes for CLI output
}

func (h *CLIHandler) WithGroup(name string) slog.Handler {
	return h // Ignore groups for CLI output
}

// New returns a slog.Logger with CLI-friendly output.
func New(level string, noColor bool) (*slog.Logger, error) {
	l := strings.ToLower(level)
	var lvl slog.Level
	switch l {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, errors.New("invalid log level")
	}

	h := &CLIHandler{level: lvl}
	return slog.New(h), nil
}

// With attaches the logger to the context.
func With(ctx context.Context, lg *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, lg)
}

// LoggerKey returns the context key for the logger.
func LoggerKey() contextKey {
	return loggerKey
}
