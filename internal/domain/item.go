package domain

import (
	"time"

	"github.com/google/uuid"
)

type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
	PriorityUrgent Priority = 4
	PriorityMust   Priority = 5
)

func (p Priority) Valid() bool {
	return p >= PriorityLow && p <= PriorityMust
}

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