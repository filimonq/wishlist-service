package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	httpadapter "github.com/filimonq/wishlist-service/internal/adapter/in/http"
	"github.com/filimonq/wishlist-service/internal/adapter/in/http/demo"
	metricsadapter "github.com/filimonq/wishlist-service/internal/adapter/out/metrics"
	"github.com/filimonq/wishlist-service/internal/adapter/out/repository"
	authsvc "github.com/filimonq/wishlist-service/internal/service/auth"
	itemsvc "github.com/filimonq/wishlist-service/internal/service/item"
	wishlistsvc "github.com/filimonq/wishlist-service/internal/service/wishlist"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Router  http.Handler
	Metrics http.Handler
	DB      *pgxpool.Pool
}

const databaseConnectTimeout = 5 * time.Second

func NewDependencies(cfg *Config) (*Dependencies, error) {
	demoConfig, err := demo.FromEnv()
	if err != nil {
		return nil, fmt.Errorf("configure demo middleware: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), databaseConnectTimeout)
	defer cancel()

	db, err := pgxpool.New(ctx, cfg.DBConn)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}

	if err = db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	userRepo := repository.NewUserRepository(db)
	wishlistRepo := repository.NewWishlistRepository(db)
	itemRepo := repository.NewItemRepository(db)

	metrics := metricsadapter.New()
	authService := authsvc.New(userRepo, cfg.JWTSecret, metrics)
	wishlistService := wishlistsvc.New(wishlistRepo, metrics)
	itemService := itemsvc.New(wishlistRepo, itemRepo, metrics)

	server := httpadapter.NewServer(authService, wishlistService, itemService, cfg.JWTSecret, db.Ping, demo.Middleware(demoConfig))

	router := server.Router(httpadapter.RequestMetrics(metrics.ObserveHTTPRequest))

	return &Dependencies{
		Router:  router,
		Metrics: metrics.Handler(),
		DB:      db,
	}, nil
}
