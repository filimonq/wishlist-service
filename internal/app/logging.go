package app

import (
	"log/slog"
	"os"
)

func ConfigureLogging() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) != 0 {
				return attr
			}
			switch attr.Key {
			case slog.TimeKey:
				return slog.Time("timestamp", attr.Value.Time().UTC())
			case slog.MessageKey:
				attr.Key = "message"
			}
			return attr
		},
	})
	slog.SetDefault(slog.New(handler).With(
		"service.name", "wishlist-service",
		"service.version", getEnv("APP_VERSION", "dev"),
		"deployment.environment.name", getEnv("APP_ENV", "development"),
	))
}
