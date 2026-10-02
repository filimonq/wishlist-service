package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	httpadapter "github.com/filimonq/wishlist-service/internal/adapter/in/http"
	"github.com/filimonq/wishlist-service/internal/adapter/out/repository"
	authsvc "github.com/filimonq/wishlist-service/internal/service/auth"
	itemsvc "github.com/filimonq/wishlist-service/internal/service/item"
	wishlistsvc "github.com/filimonq/wishlist-service/internal/service/wishlist"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Router http.Handler
	DB     *pgxpool.Pool
}

const databaseConnectTimeout = 5 * time.Second

func NewDependencies(cfg *Config) (*Dependencies, error) {
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

	authService := authsvc.New(userRepo, cfg.JWTSecret)
	wishlistService := wishlistsvc.New(wishlistRepo)
	itemService := itemsvc.New(wishlistRepo, itemRepo)

	server := httpadapter.NewServer(authService, wishlistService, itemService, cfg.JWTSecret)

	return &Dependencies{
		Router: server.Router(),
		DB:     db,
	}, nil
}
