package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Vilsol/slox"
	"github.com/lmittmann/tint"
)

func Setup() context.Context {
	level := slog.LevelInfo
	if env := os.Getenv("KLADOS_LOG_LEVEL"); env != "" {
		switch strings.ToLower(env) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return slox.Into(context.Background(), logger)
}
