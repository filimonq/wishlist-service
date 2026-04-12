package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/filimonq/wishlist-service/internal/domain"
	authsvc "github.com/filimonq/wishlist-service/internal/service/auth"
	"github.com/filimonq/wishlist-service/internal/service/mocks"
)

const testJWTSecret = "test-secret"

var errDB = errors.New("db unavailable")

func bcryptHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func TestRegister(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		email      string
		password   string
		setupMock  func(repo *mocks.UserRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:     "success",
			email:    "user@example.com",
			password: "password123",
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(nil, domain.ErrUserNotFound)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(nil)
			},
		},
		{
			name:     "user already exists",
			email:    "exists@example.com",
			password: "password123",
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("GetByEmail", mock.Anything, "exists@example.com").
					Return(&domain.User{Email: "exists@example.com"}, nil)
			},
			wantErr: domain.ErrUserAlreadyExists,
		},
		{
			name:     "repository GetByEmail error",
			email:    "user@example.com",
			password: "password123",
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(nil, errDB)
			},
			wantAnyErr: true,
		},
		{
			name:     "repository Create error",
			email:    "new@example.com",
			password: "password123",
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("GetByEmail", mock.Anything, "new@example.com").
					Return(nil, domain.ErrUserNotFound)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(errDB)
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			repo := mocks.NewUserRepository(t)
			tt.setupMock(repo)
			svc := authsvc.New(repo, testJWTSecret)

			// Act
			user, err := svc.Register(context.Background(), tt.email, tt.password)

			// Assert
			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
			case tt.wantAnyErr:
				require.Error(t, err)
				assert.Nil(t, user)
			default:
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
				assert.NotEmpty(t, user.ID)
				assert.NotEqual(t, tt.password, user.PasswordHash)
				assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.password)))
			}
		})
	}
}

func TestLogin(t *testing.T) {
	t.Parallel()

	const rawPassword = "password123"

	tests := []struct {
		name       string
		email      string
		password   string
		setupMock  func(t *testing.T, repo *mocks.UserRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:     "success",
			email:    "user@example.com",
			password: rawPassword,
			setupMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(&domain.User{Email: "user@example.com", PasswordHash: bcryptHash(t, rawPassword)}, nil)
			},
		},
		{
			name:     "user not found returns ErrInvalidCredentials",
			email:    "ghost@example.com",
			password: rawPassword,
			setupMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.On("GetByEmail", mock.Anything, "ghost@example.com").
					Return(nil, domain.ErrUserNotFound)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:     "wrong password returns ErrInvalidCredentials",
			email:    "user@example.com",
			password: "wrongpassword",
			setupMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(&domain.User{Email: "user@example.com", PasswordHash: bcryptHash(t, rawPassword)}, nil)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:     "repository error",
			email:    "user@example.com",
			password: rawPassword,
			setupMock: func(t *testing.T, repo *mocks.UserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(nil, errDB)
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			repo := mocks.NewUserRepository(t)
			tt.setupMock(t, repo)
			svc := authsvc.New(repo, testJWTSecret)

			// Act
			token, err := svc.Login(context.Background(), tt.email, tt.password)

			// Assert
			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, token)
			case tt.wantAnyErr:
				require.Error(t, err)
				assert.Empty(t, token)
			default:
				require.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}
