// @title           Wishlist API
// @version         1.0
// @description     REST API для управления вишлистами подарков
// @host            localhost:8080
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
	cfg := app.NewConfig()

	deps, err := app.NewDependencies(cfg)
	if err != nil {
		slog.Error("failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	app.New(cfg, deps).Run()
}
