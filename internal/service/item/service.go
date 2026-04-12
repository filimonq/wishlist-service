package item

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/filimonq/wishlist-service/internal/domain"
	"github.com/filimonq/wishlist-service/internal/service"
)

type Service struct {
	wishlists service.WishlistRepository
	items     service.ItemRepository
}

func New(wishlists service.WishlistRepository, items service.ItemRepository) service.ItemService {
	return &Service{wishlists: wishlists, items: items}
}

func (s *Service) Create(ctx context.Context, userID, wishlistID uuid.UUID, name, description, url string, priority domain.Priority) (*domain.Item, error) {
	if !priority.Valid() {
		return nil, domain.ErrInvalidPriority
	}

	w, err := s.wishlists.GetByID(ctx, wishlistID)
	if err != nil {
		return nil, fmt.Errorf("get wishlist: %w", err)
	}
	if w.UserID != userID {
		return nil, domain.ErrWishlistAccessDenied
	}

	item := &domain.Item{
		ID:          uuid.New(),
		WishlistID:  wishlistID,
		Name:        name,
		Description: description,
		URL:         url,
		Priority:    priority,
		IsReserved:  false,
	}

	if err = s.items.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create item: %w", err)
	}

	return item, nil
}

func (s *Service) List(ctx context.Context, userID, wishlistID uuid.UUID) ([]*domain.Item, error) {
	w, err := s.wishlists.GetByID(ctx, wishlistID)
	if err != nil {
		return nil, fmt.Errorf("get wishlist: %w", err)
	}
	if w.UserID != userID {
		return nil, domain.ErrWishlistAccessDenied
	}

	items, err := s.items.ListByWishlistID(ctx, wishlistID)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}

	return items, nil
}

func (s *Service) ListByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]*domain.Item, error) {
	items, err := s.items.ListByWishlistID(ctx, wishlistID)
	if err != nil {
		return nil, fmt.Errorf("list items by wishlist id: %w", err)
	}

	return items, nil
}

func (s *Service) Update(ctx context.Context, userID, wishlistID, itemID uuid.UUID, name, description, url *string, priority *domain.Priority) (*domain.Item, error) {
	w, err := s.wishlists.GetByID(ctx, wishlistID)
	if err != nil {
		return nil, fmt.Errorf("get wishlist: %w", err)
	}
	if w.UserID != userID {
		return nil, domain.ErrWishlistAccessDenied
	}

	item, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	if item.WishlistID != wishlistID {
		return nil, domain.ErrItemNotInWishlist
	}

	if name != nil {
		item.Name = *name
	}
	if description != nil {
		item.Description = *description
	}
	if url != nil {
		item.URL = *url
	}
	if priority != nil {
		if !priority.Valid() {
			return nil, domain.ErrInvalidPriority
		}
		item.Priority = *priority
	}

	if err = s.items.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}

	return item, nil
}

func (s *Service) Delete(ctx context.Context, userID, wishlistID, itemID uuid.UUID) error {
	w, err := s.wishlists.GetByID(ctx, wishlistID)
	if err != nil {
		return fmt.Errorf("get wishlist: %w", err)
	}
	if w.UserID != userID {
		return domain.ErrWishlistAccessDenied
	}

	item, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("get item: %w", err)
	}
	if item.WishlistID != wishlistID {
		return domain.ErrItemNotInWishlist
	}

	if err = s.items.Delete(ctx, itemID); err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	return nil
}

func (s *Service) Reserve(ctx context.Context, token uuid.UUID, itemID uuid.UUID) error {
	w, err := s.wishlists.GetByPublicToken(ctx, token)
	if err != nil {
		return fmt.Errorf("get wishlist by token: %w", err)
	}

	item, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("get item: %w", err)
	}
	if item.WishlistID != w.ID {
		return domain.ErrItemNotInWishlist
	}

	if err = s.items.Reserve(ctx, itemID); err != nil {
		return fmt.Errorf("reserve item: %w", err)
	}

	return nil
}
