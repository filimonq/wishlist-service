package domain

import (
	"time"

	"github.com/google/uuid"
)

type Wishlist struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description string
	EventDate   time.Time
	PublicToken uuid.UUID
}
