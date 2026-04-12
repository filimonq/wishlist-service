package domain

import (
	"time"

	"github.com/google/uuid"
)

type Priority int

type Item struct {
	ID          uuid.UUID
	WishlistID  uuid.UUID
	Name        string
	Description string
	URL         string
	Priority    Priority
	IsReserved  bool
	ReservedAt  *time.Time
}
