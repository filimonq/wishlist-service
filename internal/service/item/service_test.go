package item_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	metricsadapter "github.com/filimonq/wishlist-service/internal/adapter/out/metrics"
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
		priority   domain.Priority
		reason     string
	}{
		{
			name:     "success",
			userID:   testUserID,
			priority: domain.Priority(3),
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("Create", mock.Anything, mock.AnythingOfType("*domain.Item")).Return(nil)
			},
		},
		{
			name:     "wishlist not found",
			userID:   testUserID,
			priority: domain.Priority(3),
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).
					Return(nil, domain.ErrWishlistNotFound)
			},
			wantErr: domain.ErrWishlistNotFound,
			reason:  "wishlist_not_found",
		},
		{
			name:     "access denied",
			userID:   otherUserID,
			priority: domain.Priority(3),
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
			},
			wantErr: domain.ErrWishlistAccessDenied,
			reason:  "access_denied",
		},
		{
			name:     "repository create error",
			userID:   testUserID,
			priority: domain.Priority(3),
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(newWishlist(), nil)
				it.On("Create", mock.Anything, mock.AnythingOfType("*domain.Item")).Return(errDB)
			},
			wantAnyErr: true,
			reason:     "internal_error",
		},
		{
			name:      "invalid priority",
			userID:    testUserID,
			priority:  domain.Priority(0),
			setupMock: func(_ *mocks.WishlistRepository, _ *mocks.ItemRepository) {},
			wantErr:   domain.ErrInvalidPriority,
			reason:    "invalid_priority",
		},
		{
			name:     "wishlist repository error",
			userID:   testUserID,
			priority: domain.Priority(3),
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByID", mock.Anything, testWishlistID).Return(nil, errDB)
			},
			wantAnyErr: true,
			reason:     "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			wlRepo := mocks.NewWishlistRepository(t)
			itRepo := mocks.NewItemRepository(t)
			tt.setupMock(wlRepo, itRepo)
			metrics := metricsadapter.New()
			svc := itemsvc.New(wlRepo, itRepo, metrics)

			// Act
			item, err := svc.Create(context.Background(), tt.userID, testWishlistID, "PS5", "", "", tt.priority)
			recorder := httptest.NewRecorder()
			metrics.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
			value := "0"
			if err == nil {
				value = "1"
			}
			assert.Contains(t, recorder.Body.String(), "wishlist_items_created_total "+value+"\n")
			for _, reason := range []string{"invalid_priority", "wishlist_not_found", "access_denied", "internal_error"} {
				failureValue := "0"
				if reason == tt.reason {
					failureValue = "1"
				}
				assert.Contains(t, recorder.Body.String(),
					`wishlist_item_creation_failures_total{reason="`+reason+`"} `+failureValue+"\n")
			}

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
			svc := itemsvc.New(wlRepo, itRepo, nil)

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
			svc := itemsvc.New(wlRepo, itRepo, nil)

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
		reason    string
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
			reason:  "wishlist_not_found",
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
			reason:  "item_not_in_wishlist",
		},
		{
			name: "already reserved",
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(newItem(), nil)
				it.On("Reserve", mock.Anything, testItemID).Return(domain.ErrItemAlreadyReserved)
			},
			wantErr: domain.ErrItemAlreadyReserved,
			reason:  "already_reserved",
		},
		{
			name: "item not found",
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(nil, domain.ErrItemNotFound)
			},
			wantErr: domain.ErrItemNotFound,
			reason:  "item_not_found",
		},
		{
			name: "wishlist repository error",
			setupMock: func(wl *mocks.WishlistRepository, _ *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).Return(nil, errDB)
			},
			wantErr: errDB,
			reason:  "internal_error",
		},
		{
			name: "item repository error",
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(nil, errDB)
			},
			wantErr: errDB,
			reason:  "internal_error",
		},
		{
			name: "reserve repository error",
			setupMock: func(wl *mocks.WishlistRepository, it *mocks.ItemRepository) {
				wl.On("GetByPublicToken", mock.Anything, testPublicToken).Return(newWishlist(), nil)
				it.On("GetByID", mock.Anything, testItemID).Return(newItem(), nil)
				it.On("Reserve", mock.Anything, testItemID).Return(errDB)
			},
			wantErr: errDB,
			reason:  "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			wlRepo := mocks.NewWishlistRepository(t)
			itRepo := mocks.NewItemRepository(t)
			tt.setupMock(wlRepo, itRepo)
			metrics := metricsadapter.New()
			svc := itemsvc.New(wlRepo, itRepo, metrics)

			// Act
			err := svc.Reserve(context.Background(), testPublicToken, testItemID)
			recorder := httptest.NewRecorder()
			metrics.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
			value := "0"
			if err == nil {
				value = "1"
			}
			assert.Contains(t, recorder.Body.String(), "wishlist_items_reserved_total "+value+"\n")
			for _, reason := range []string{"wishlist_not_found", "item_not_found", "item_not_in_wishlist", "already_reserved", "internal_error"} {
				failureValue := "0"
				if tt.reason == reason {
					failureValue = "1"
				}
				assert.Contains(t, recorder.Body.String(),
					`wishlist_item_reservation_failures_total{reason="`+reason+`"} `+failureValue+"\n")
			}

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
