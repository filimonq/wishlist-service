package main

import (
	"log/slog"
	"os"

	"github.com/filimonq/wishlist-service/internal/app"
)

func main() {
	cfg := app.NewConfig()

	deps, err := app.NewDependencies(cfg)
	if err != nil {
		slog.Error("failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	app.New(cfg, deps).Run()
}
