package domain

import "errors"

var (
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrItemAlreadyReserved = errors.New("item already reserved")

	// еще подумать
)
