package logger

import (
	"fmt"
	"log/slog"
	"os"
)

const (
	// EnvLocal configures local development logging.
	EnvLocal = "local"
	// EnvProd configures production logging.
	EnvProd = "prod"
)

// Setup builds a logger configured for the given environment.
func Setup(env string) *slog.Logger {
	switch env {
	case EnvLocal:
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case EnvProd:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		panic(fmt.Sprintf("unknown environment: %s", env))
	}
}
