package app

import (
	"context"
	"fmt"
	"net/http"

	httpadapter "github.com/filimonq/wishlist-service/internal/adapter/in/http"
	"github.com/filimonq/wishlist-service/internal/adapter/out/repository"
	"github.com/filimonq/wishlist-service/internal/migrate"
	authsvc "github.com/filimonq/wishlist-service/internal/service/auth"
	itemsvc "github.com/filimonq/wishlist-service/internal/service/item"
	wishlistsvc "github.com/filimonq/wishlist-service/internal/service/wishlist"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Router http.Handler
	DB     *pgxpool.Pool
}

func NewDependencies(cfg *Config) (*Dependencies, error) {
	if err := migrate.Run(cfg.DBConn, cfg.MigrationsPath); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	db, err := pgxpool.New(context.Background(), cfg.DBConn)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}

	if err = db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	userRepo := repository.NewUserRepository(db)
	wishlistRepo := repository.NewWishlistRepository(db)
	itemRepo := repository.NewItemRepository(db)

	authService := authsvc.New(userRepo, cfg.JWTSecret)
	wishlistService := wishlistsvc.New(wishlistRepo)
	itemService := itemsvc.New(wishlistRepo, itemRepo)

	server := httpadapter.NewServer(authService, wishlistService, itemService, cfg.JWTSecret)

	return &Dependencies{
		Router: server.Router(),
		DB:     db,
	}, nil
}
