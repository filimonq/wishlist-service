package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/filimonq/wishlist-service/internal/domain"
	"github.com/filimonq/wishlist-service/internal/service"
)

type Service struct {
	users     service.UserRepository
	jwtSecret string
}

func New(users service.UserRepository, jwtSecret string) service.AuthService {
	return &Service{users: users, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, email, password string) (*domain.User, error) {
	existing, err := s.users.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
	}

	if err = s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", fmt.Errorf("get user by email: %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", domain.ErrInvalidCredentials
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID.String(),
	})

	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}
