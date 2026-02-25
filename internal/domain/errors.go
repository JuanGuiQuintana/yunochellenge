package domain

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrDuplicate         = errors.New("duplicate")
	ErrInvalidInput      = errors.New("invalid input")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrFXNotAvailable    = errors.New("fx rate not available")
)
