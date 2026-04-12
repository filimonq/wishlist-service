package service

import (
	"context"
	"time"

	"github.com/filimonq/wishlist-service/internal/domain"
	"github.com/google/uuid"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (token string, err error)
}

type WishlistService interface {
	Create(ctx context.Context, userID uuid.UUID, title, description string, eventDate time.Time) (*domain.Wishlist, error)
	List(ctx context.Context, userID uuid.UUID) ([]*domain.Wishlist, error)
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Wishlist, error)
	Update(ctx context.Context, userID, id uuid.UUID, title, description *string, eventDate *time.Time) (*domain.Wishlist, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	GetByPublicToken(ctx context.Context, token uuid.UUID) (*domain.Wishlist, error)
}

type ItemService interface {
	Create(ctx context.Context, userID, wishlistID uuid.UUID, name, description, url string, priority domain.Priority) (*domain.Item, error)
	List(ctx context.Context, userID, wishlistID uuid.UUID) ([]*domain.Item, error)
	ListByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]*domain.Item, error)
	Update(ctx context.Context, userID, wishlistID, itemID uuid.UUID, name, description, url *string, priority *domain.Priority) (*domain.Item, error)
	Delete(ctx context.Context, userID, wishlistID, itemID uuid.UUID) error
	Reserve(ctx context.Context, token uuid.UUID, itemID uuid.UUID) error
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type WishlistRepository interface {
	Create(ctx context.Context, wishlist *domain.Wishlist) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error)
	GetByPublicToken(ctx context.Context, token uuid.UUID) (*domain.Wishlist, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Wishlist, error)
	Update(ctx context.Context, wishlist *domain.Wishlist) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error)
	ListByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) error
	Delete(ctx context.Context, id uuid.UUID) error
	Reserve(ctx context.Context, id uuid.UUID) error
}
