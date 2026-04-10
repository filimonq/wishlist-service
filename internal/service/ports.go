package service

import (
	"context"

	"github.com/filimonq/wishlist-service/internal/domain"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) // подумать да
}

// по интерфейсу на каждый сервис и репозиторий короче
