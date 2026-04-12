package wishlist_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/filimonq/wishlist-service/internal/domain"
	"github.com/filimonq/wishlist-service/internal/service/mocks"
	wishlistsvc "github.com/filimonq/wishlist-service/internal/service/wishlist"
)

var (
	testUserID     = uuid.New()
	testWishlistID = uuid.New()
	testEventDate  = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	errDB          = errors.New("db unavailable")
)

func newWishlist() *domain.Wishlist {
	return &domain.Wishlist{
		ID:          testWishlistID,
		UserID:      testUserID,
		Title:       "Birthday",
		Description: "My birthday wishlist",
		EventDate:   testEventDate,
		PublicToken: uuid.New(),
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(repo *mocks.WishlistRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "success",
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Wishlist")).
					Return(nil)
			},
		},
		{
			name: "repository error",
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Wishlist")).
					Return(errDB)
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			repo := mocks.NewWishlistRepository(t)
			tt.setupMock(repo)
			svc := wishlistsvc.New(repo)

			// Act
			w, err := svc.Create(context.Background(), testUserID, "Birthday", "My birthday wishlist", testEventDate)

			// Assert
			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, w)
			case tt.wantAnyErr:
				require.Error(t, err)
				assert.Nil(t, w)
			default:
				require.NoError(t, err)
				require.NotNil(t, w)
				assert.Equal(t, testUserID, w.UserID)
				assert.NotEqual(t, uuid.Nil, w.ID)
				assert.NotEqual(t, uuid.Nil, w.PublicToken)
			}
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	otherUserID := uuid.New()

	tests := []struct {
		name      string
		userID    uuid.UUID
		setupMock func(repo *mocks.WishlistRepository)
		wantErr   error
	}{
		{
			name:   "success",
			userID: testUserID,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
			},
		},
		{
			name:   "access denied for other user",
			userID: otherUserID,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
			},
			wantErr: domain.ErrWishlistAccessDenied,
		},
		{
			name:   "not found",
			userID: testUserID,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).
					Return(nil, domain.ErrWishlistNotFound)
			},
			wantErr: domain.ErrWishlistNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			repo := mocks.NewWishlistRepository(t)
			tt.setupMock(repo)
			svc := wishlistsvc.New(repo)

			// Act
			w, err := svc.Get(context.Background(), tt.userID, testWishlistID)

			// Assert
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, w)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testWishlistID, w.ID)
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	newTitle := "New Title"
	newDesc := "New Description"
	newDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	otherUserID := uuid.New()

	tests := []struct {
		name       string
		userID     uuid.UUID
		title      *string
		desc       *string
		date       *time.Time
		setupMock  func(repo *mocks.WishlistRepository)
		wantErr    error
		wantAnyErr bool
		check      func(t *testing.T, w *domain.Wishlist)
	}{
		{
			name:   "success — full update",
			userID: testUserID,
			title:  &newTitle,
			desc:   &newDesc,
			date:   &newDate,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Wishlist")).Return(nil)
			},
			check: func(t *testing.T, w *domain.Wishlist) {
				t.Helper()
				assert.Equal(t, newTitle, w.Title)
				assert.Equal(t, newDesc, w.Description)
				assert.Equal(t, newDate, w.EventDate)
			},
		},
		{
			name:   "success — partial update only title",
			userID: testUserID,
			title:  &newTitle,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Wishlist")).Return(nil)
			},
			check: func(t *testing.T, w *domain.Wishlist) {
				t.Helper()
				assert.Equal(t, newTitle, w.Title)
				assert.Equal(t, "My birthday wishlist", w.Description) // не изменилось
			},
		},
		{
			name:   "access denied",
			userID: otherUserID,
			title:  &newTitle,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
			},
			wantErr: domain.ErrWishlistAccessDenied,
		},
		{
			name:   "not found",
			userID: testUserID,
			title:  &newTitle,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).
					Return(nil, domain.ErrWishlistNotFound)
			},
			wantErr: domain.ErrWishlistNotFound,
		},
		{
			name:   "repository update error",
			userID: testUserID,
			title:  &newTitle,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Wishlist")).Return(errDB)
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			repo := mocks.NewWishlistRepository(t)
			tt.setupMock(repo)
			svc := wishlistsvc.New(repo)

			// Act
			w, err := svc.Update(context.Background(), tt.userID, testWishlistID, tt.title, tt.desc, tt.date)

			// Assert
			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, w)
			case tt.wantAnyErr:
				require.Error(t, err)
				assert.Nil(t, w)
			default:
				require.NoError(t, err)
				require.NotNil(t, w)
				if tt.check != nil {
					tt.check(t, w)
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	otherUserID := uuid.New()

	tests := []struct {
		name       string
		userID     uuid.UUID
		setupMock  func(repo *mocks.WishlistRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:   "success",
			userID: testUserID,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				repo.On("Delete", mock.Anything, testWishlistID).Return(nil)
			},
		},
		{
			name:   "access denied",
			userID: otherUserID,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
			},
			wantErr: domain.ErrWishlistAccessDenied,
		},
		{
			name:   "not found",
			userID: testUserID,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).
					Return(nil, domain.ErrWishlistNotFound)
			},
			wantErr: domain.ErrWishlistNotFound,
		},
		{
			name:   "repository delete error",
			userID: testUserID,
			setupMock: func(repo *mocks.WishlistRepository) {
				repo.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				repo.On("Delete", mock.Anything, testWishlistID).Return(errDB)
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			repo := mocks.NewWishlistRepository(t)
			tt.setupMock(repo)
			svc := wishlistsvc.New(repo)

			// Act
			err := svc.Delete(context.Background(), tt.userID, testWishlistID)

			// Assert
			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			case tt.wantAnyErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}
		})
	}
}
