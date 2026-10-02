// @title           Wishlist API
// @version         1.0
// @description     REST API для управления вишлистами подарков
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
package main

import (
	"log/slog"
	"os"

	_ "github.com/filimonq/wishlist-service/docs"
	"github.com/filimonq/wishlist-service/internal/app"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := app.NewConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	deps, err := app.NewDependencies(cfg)
	if err != nil {
		slog.Error("failed to initialize dependencies", "error", cfg.RedactError(err))
		os.Exit(1)
	}

	if err = app.New(cfg, deps).Run(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
