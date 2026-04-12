package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/filimonq/wishlist-service/internal/domain"
)

type ItemRepository struct {
	db *pgxpool.Pool
}

func NewItemRepository(db *pgxpool.Pool) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO items (id, wishlist_id, name, description, url, priority)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		item.ID, item.WishlistID, item.Name, item.Description, item.URL, item.Priority,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		// 23503 — foreign_key_violation: wishlist_id не существует
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.ErrWishlistNotFound
		}
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

func (r *ItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	var item domain.Item

	err := r.db.QueryRow(ctx,
		`SELECT id, wishlist_id, name, description, url, priority, is_reserved, reserved_at
		 FROM items WHERE id = $1`,
		id,
	).Scan(
		&item.ID, &item.WishlistID, &item.Name, &item.Description,
		&item.URL, &item.Priority, &item.IsReserved, &item.ReservedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}

	return &item, nil
}

func (r *ItemRepository) ListByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]*domain.Item, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, wishlist_id, name, description, url, priority, is_reserved, reserved_at
		 FROM items WHERE wishlist_id = $1`,
		wishlistID,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var items []*domain.Item
	for rows.Next() {
		var item domain.Item
		if err = rows.Scan(
			&item.ID, &item.WishlistID, &item.Name, &item.Description,
			&item.URL, &item.Priority, &item.IsReserved, &item.ReservedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, &item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return items, nil
}

func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) error {
	result, err := r.db.Exec(ctx,
		`UPDATE items SET name = $1, description = $2, url = $3, priority = $4, updated_at = NOW()
		 WHERE id = $5`,
		item.Name, item.Description, item.URL, item.Priority, item.ID,
	)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrItemNotFound
	}

	return nil
}

func (r *ItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM items WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrItemNotFound
	}

	return nil
}

func (r *ItemRepository) Reserve(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx,
		`UPDATE items SET is_reserved = true, reserved_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND is_reserved = false`,
		id,
	)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrItemAlreadyReserved
	}

	return nil
}
