package item_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/filimonq/wishlist-service/internal/domain"
	itemsvc "github.com/filimonq/wishlist-service/internal/service/item"
	"github.com/filimonq/wishlist-service/internal/service/mocks"
)

var (
	testUserID      = uuid.New()
	testWishlistID  = uuid.New()
	testItemID      = uuid.New()
	testPublicToken = uuid.New()
	errDB           = errors.New("db unavailable")
)

func newWishlist() *domain.Wishlist {
	return &domain.Wishlist{
		ID:          testWishlistID,
		UserID:      testUserID,
		PublicToken: testPublicToken,
	}
}

func newItem() *domain.Item {
	return &domain.Item{
		ID:         testItemID,
		WishlistID: testWishlistID,
		Name:       "PlayStation 5",
		Priority:   domain.Priority(3),
		IsReserved: false,
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	otherUserID := uuid.New()

	tests := []struct {
		name       string
		userID     uuid.UUID
		setupMock  func(wl *mocks.WishlistRepository, it *mocks.ItemRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:   "success",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("Create", mock.Anything, mock.AnythingOfType("*domain.Item")).Return(nil)
			},
		},
		{
			name:   "wishlist not found",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).
					Return(nil, domain.ErrWishlistNotFound)
			},
			wantErr: domain.ErrWishlistNotFound,
		},
		{
			name:   "access denied",
			userID: otherUserID,
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
			},
			wantErr: domain.ErrWishlistAccessDenied,
		},
		{
			name:   "repository create error",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("Create", mock.Anything, mock.AnythingOfType("*domain.Item")).Return(errDB)
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			wlRepo := mocks.NewWishlistRepository(t)
			itRepo := mocks.NewItemRepository(t)
			tt.setupMock(wlRepo, itRepo)
			svc := itemsvc.New(wlRepo, itRepo)

			// Act
			item, err := svc.Create(context.Background(), tt.userID, testWishlistID, "PS5", "", "", domain.Priority(3))

			// Assert
			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, item)
			case tt.wantAnyErr:
				require.Error(t, err)
				assert.Nil(t, item)
			default:
				require.NoError(t, err)
				require.NotNil(t, item)
				assert.Equal(t, testWishlistID, item.WishlistID)
				assert.NotEqual(t, uuid.Nil, item.ID)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	newName := "Xbox Series X"
	otherUserID := uuid.New()
	otherWishlistID := uuid.New()

	tests := []struct {
		name       string
		userID     uuid.UUID
		setupMock  func(wl *mocks.WishlistRepository, it *mocks.ItemRepository)
		wantErr    error
		wantAnyErr bool
		check      func(t *testing.T, item *domain.Item)
	}{
		{
			name:   "success",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(newItem(), nil)
				it.On("Update", mock.Anything, mock.AnythingOfType("*domain.Item")).Return(nil)
			},
			check: func(t *testing.T, item *domain.Item) {
				t.Helper()
				assert.Equal(t, newName, item.Name)
			},
		},
		{
			name:   "access denied",
			userID: otherUserID,
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
			},
			wantErr: domain.ErrWishlistAccessDenied,
		},
		{
			name:   "item not in wishlist",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				wrongItem := newItem()
				wrongItem.WishlistID = otherWishlistID
				it.On("GetByID", mock.Anything, testItemID).Return(wrongItem, nil)
			},
			wantErr: domain.ErrItemNotInWishlist,
		},
		{
			name:   "item not found",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(nil, domain.ErrItemNotFound)
			},
			wantErr: domain.ErrItemNotFound,
		},
		{
			name:   "repository update error",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(newItem(), nil)
				it.On("Update", mock.Anything, mock.AnythingOfType("*domain.Item")).Return(errDB)
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			wlRepo := mocks.NewWishlistRepository(t)
			itRepo := mocks.NewItemRepository(t)
			tt.setupMock(wlRepo, itRepo)
			svc := itemsvc.New(wlRepo, itRepo)

			// Act
			item, err := svc.Update(context.Background(), tt.userID, testWishlistID, testItemID, &newName, nil, nil, nil)

			// Assert
			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, item)
			case tt.wantAnyErr:
				require.Error(t, err)
				assert.Nil(t, item)
			default:
				require.NoError(t, err)
				require.NotNil(t, item)
				if tt.check != nil {
					tt.check(t, item)
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	otherUserID := uuid.New()
	otherWishlistID := uuid.New()

	tests := []struct {
		name       string
		userID     uuid.UUID
		setupMock  func(wl *mocks.WishlistRepository, it *mocks.ItemRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:   "success",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(newItem(), nil)
				it.On("Delete", mock.Anything, testItemID).Return(nil)
			},
		},
		{
			name:   "access denied",
			userID: otherUserID,
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
			},
			wantErr: domain.ErrWishlistAccessDenied,
		},
		{
			name:   "item not in wishlist",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				wrongItem := newItem()
				wrongItem.WishlistID = otherWishlistID
				it.On("GetByID", mock.Anything, testItemID).Return(wrongItem, nil)
			},
			wantErr: domain.ErrItemNotInWishlist,
		},
		{
			name:   "repository delete error",
			userID: testUserID,
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(newItem(), nil)
				it.On("Delete", mock.Anything, testItemID).Return(errDB)
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			wlRepo := mocks.NewWishlistRepository(t)
			itRepo := mocks.NewItemRepository(t)
			tt.setupMock(wlRepo, itRepo)
			svc := itemsvc.New(wlRepo, itRepo)

			// Act
			err := svc.Delete(context.Background(), tt.userID, testWishlistID, testItemID)

			// Assert
			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantAnyErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestReserve(t *testing.T) {
	t.Parallel()

	otherWishlistID := uuid.New()

	tests := []struct {
		name      string
		setupMock func(wl *mocks.WishlistRepository, it *mocks.ItemRepository)
		wantErr   error
	}{
		{
			name: "success",
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(newItem(), nil)
				it.On("Reserve", mock.Anything, testItemID).Return(nil)
			},
		},
		{
			name: "wishlist not found by token",
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).
					Return(nil, domain.ErrWishlistNotFound)
			},
			wantErr: domain.ErrWishlistNotFound,
		},
		{
			name: "item not in wishlist",
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).Return(newWishlist(), nil)
				wrongItem := newItem()
				wrongItem.WishlistID = otherWishlistID
				it.On("GetByID", mock.Anything, testItemID).Return(wrongItem, nil)
			},
			wantErr: domain.ErrItemNotInWishlist,
		},
		{
			name: "already reserved",
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(newItem(), nil)
				it.On("Reserve", mock.Anything, testItemID).Return(domain.ErrItemAlreadyReserved)
			},
			wantErr: domain.ErrItemAlreadyReserved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			wlRepo := mocks.NewWishlistRepository(t)
			itRepo := mocks.NewItemRepository(t)
			tt.setupMock(wlRepo, itRepo)
			svc := itemsvc.New(wlRepo, itRepo)

			// Act
			err := svc.Reserve(context.Background(), testPublicToken, testItemID)

			// Assert
			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
