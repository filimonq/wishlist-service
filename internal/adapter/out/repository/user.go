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

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		user.ID, user.Email, user.PasswordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		// 23505 — unique_violation: email уже занят
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User

	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}

	return &user, nil
}
