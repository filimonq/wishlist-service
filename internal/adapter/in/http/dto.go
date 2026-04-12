package httpadapter

import (
	"time"

	"github.com/google/uuid"

	"github.com/filimonq/wishlist-service/internal/domain"
)

type RegisterRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type UserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type CreateWishlistRequest struct {
	Title       string    `json:"title"       binding:"required,min=1,max=255"`
	Description string    `json:"description" binding:"max=1000"`
	EventDate   time.Time `json:"event_date"  binding:"required"`
}

type UpdateWishlistRequest struct {
	Title       *string    `json:"title"       binding:"omitempty,min=1,max=255"`
	Description *string    `json:"description" binding:"omitempty,max=1000"`
	EventDate   *time.Time `json:"event_date"`
}

type WishlistResponse struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	EventDate   time.Time      `json:"event_date"`
	PublicToken uuid.UUID      `json:"public_token"`
	Items       []ItemResponse `json:"items,omitempty"`
}

type CreateItemRequest struct {
	Name        string          `json:"name"        binding:"required,min=1,max=255"`
	Description string          `json:"description" binding:"max=1000"`
	URL         string          `json:"url"         binding:"omitempty,url"`
	Priority    domain.Priority `json:"priority"    binding:"required,min=1,max=5"`
}

type UpdateItemRequest struct {
	Name        *string          `json:"name"        binding:"omitempty,min=1,max=255"`
	Description *string          `json:"description" binding:"omitempty,max=1000"`
	URL         *string          `json:"url"         binding:"omitempty,url"`
	Priority    *domain.Priority `json:"priority"    binding:"omitempty,min=1,max=5"`
}

type ItemResponse struct {
	ID          uuid.UUID       `json:"id"`
	WishlistID  uuid.UUID       `json:"wishlist_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	URL         string          `json:"url"`
	Priority    domain.Priority `json:"priority"`
	IsReserved  bool            `json:"is_reserved"`
	ReservedAt  *time.Time      `json:"reserved_at,omitempty"`
}

func toWishlistResponse(w *domain.Wishlist) WishlistResponse {
	return WishlistResponse{
		ID:          w.ID,
		UserID:      w.UserID,
		Title:       w.Title,
		Description: w.Description,
		EventDate:   w.EventDate,
		PublicToken: w.PublicToken,
	}
}

func toItemResponse(item *domain.Item) ItemResponse {
	return ItemResponse{
		ID:          item.ID,
		WishlistID:  item.WishlistID,
		Name:        item.Name,
		Description: item.Description,
		URL:         item.URL,
		Priority:    item.Priority,
		IsReserved:  item.IsReserved,
		ReservedAt:  item.ReservedAt,
	}
}

func toItemResponses(items []*domain.Item) []ItemResponse {
	result := make([]ItemResponse, len(items))
	for i, item := range items {
		result[i] = toItemResponse(item)
	}
	return result
}
