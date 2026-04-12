package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/filimonq/wishlist-service/internal/domain"
)

type WishlistRepository struct {
	db *pgxpool.Pool
}

func NewWishlistRepository(db *pgxpool.Pool) *WishlistRepository {
	return &WishlistRepository{db: db}
}

func (r *WishlistRepository) Create(ctx context.Context, w *domain.Wishlist) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO wishlists (id, user_id, title, description, event_date, public_token)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		w.ID, w.UserID, w.Title, w.Description, w.EventDate, w.PublicToken,
	)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

func (r *WishlistRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
	var w domain.Wishlist

	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, description, event_date, public_token
		 FROM wishlists WHERE id = $1`,
		id,
	).Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.EventDate, &w.PublicToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWishlistNotFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}

	return &w, nil
}

func (r *WishlistRepository) GetByPublicToken(ctx context.Context, token uuid.UUID) (*domain.Wishlist, error) {
	var w domain.Wishlist

	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, description, event_date, public_token
		 FROM wishlists WHERE public_token = $1`,
		token,
	).Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.EventDate, &w.PublicToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWishlistNotFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}

	return &w, nil
}

func (r *WishlistRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Wishlist, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, description, event_date, public_token
		 FROM wishlists WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var wishlists []*domain.Wishlist
	for rows.Next() {
		var w domain.Wishlist
		if err = rows.Scan(
			&w.ID, &w.UserID, &w.Title, &w.Description, &w.EventDate, &w.PublicToken,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		wishlists = append(wishlists, &w)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return wishlists, nil
}

func (r *WishlistRepository) Update(ctx context.Context, w *domain.Wishlist) error {
	result, err := r.db.Exec(ctx,
		`UPDATE wishlists SET title = $1, description = $2, event_date = $3, updated_at = NOW()
		 WHERE id = $4`,
		w.Title, w.Description, w.EventDate, w.ID,
	)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrWishlistNotFound
	}

	return nil
}

func (r *WishlistRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM wishlists WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrWishlistNotFound
	}

	return nil
}
