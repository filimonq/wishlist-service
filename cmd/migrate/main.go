package main

import (
	"log/slog"
	"os"

	"github.com/filimonq/wishlist-service/internal/app"
	"github.com/filimonq/wishlist-service/internal/migrate"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := app.NewMigrationConfig()
	if err != nil {
		slog.Error("invalid migration configuration", "error", err)
		os.Exit(1)
	}
	if err = migrate.Run(cfg.DBConn, cfg.MigrationsPath); err != nil {
		slog.Error("migration failed", "error", cfg.RedactError(err))
		os.Exit(1)
	}
	slog.Info("migrations completed")
}
