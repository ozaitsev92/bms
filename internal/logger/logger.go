// Package logger builds the application's structured slog logger.
package logger

import (
	"fmt"
	"io"
	"log/slog"
)

const (
	// EnvLocal configures human-readable debug logging.
	EnvLocal = "local"
	// EnvProd configures JSON logging at info level.
	EnvProd = "prod"
)

// New builds a logger writing to w, configured for the given environment.
func New(env string, w io.Writer) (*slog.Logger, error) {
	switch env {
	case EnvLocal:
		return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})), nil
	case EnvProd:
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})), nil
	default:
		return nil, fmt.Errorf("unknown environment %q (want %q or %q)", env, EnvLocal, EnvProd)
	}
}
