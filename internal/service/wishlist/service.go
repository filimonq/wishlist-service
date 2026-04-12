package wishlist

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/filimonq/wishlist-service/internal/domain"
	"github.com/filimonq/wishlist-service/internal/service"
)

type Service struct {
	wishlists service.WishlistRepository
}

func New(wishlists service.WishlistRepository) service.WishlistService {
	return &Service{wishlists: wishlists}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, title, description string, eventDate time.Time) (*domain.Wishlist, error) {
	w := &domain.Wishlist{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       title,
		Description: description,
		EventDate:   eventDate,
		PublicToken: uuid.New(),
	}

	if err := s.wishlists.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("create wishlist: %w", err)
	}

	return w, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]*domain.Wishlist, error) {
	wishlists, err := s.wishlists.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list wishlists: %w", err)
	}

	return wishlists, nil
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Wishlist, error) {
	w, err := s.wishlists.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get wishlist: %w", err)
	}

	if w.UserID != userID {
		return nil, domain.ErrWishlistAccessDenied
	}

	return w, nil
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, title, description *string, eventDate *time.Time) (*domain.Wishlist, error) {
	w, err := s.wishlists.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get wishlist: %w", err)
	}

	if w.UserID != userID {
		return nil, domain.ErrWishlistAccessDenied
	}

	if title != nil {
		w.Title = *title
	}
	if description != nil {
		w.Description = *description
	}
	if eventDate != nil {
		w.EventDate = *eventDate
	}

	if err = s.wishlists.Update(ctx, w); err != nil {
		return nil, fmt.Errorf("update wishlist: %w", err)
	}

	return w, nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	w, err := s.wishlists.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get wishlist: %w", err)
	}

	if w.UserID != userID {
		return domain.ErrWishlistAccessDenied
	}

	if err = s.wishlists.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete wishlist: %w", err)
	}

	return nil
}

func (s *Service) GetByPublicToken(ctx context.Context, token uuid.UUID) (*domain.Wishlist, error) {
	w, err := s.wishlists.GetByPublicToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get wishlist by token: %w", err)
	}

	return w, nil
}
