package domain

import "errors"

var (
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrItemAlreadyReserved  = errors.New("item already reserved")
	ErrItemNotInWishlist    = errors.New("item does not belong to wishlist")
	ErrWishlistNotFound     = errors.New("wishlist not found")
	ErrItemNotFound         = errors.New("item not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrWishlistAccessDenied = errors.New("wishlist access denied")
	ErrEventDateInPast      = errors.New("event date must be in the future")
	ErrInvalidPriority      = errors.New("priority must be between 1 and 5")
)
