package utils

import (
	"log/slog"
	"os"
)

func SetupLogger(env string) {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

func LogAction(user, action, registry, repository, digest, result string, err error) {
	attrs := []any{
		"user", user,
		"action", action,
		"registry", registry,
		"repository", repository,
		"digest", digest,
		"result", result,
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
		slog.Error("operation", attrs...)
		return
	}
	slog.Info("operation", attrs...)
}
