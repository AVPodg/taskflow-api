package auth

import "errors"

var (
	// repository error
	ErrForeignKey             = errors.New("error foreign key")
	ErrUnique                 = errors.New("error unique")
	ErrInvalidAmount          = errors.New("invalid amount")
	ErrProductNotFound        = errors.New("product not found")
	ErrNotEnoughStock         = errors.New("stock < amount")
	ErrUserNotFound           = errors.New("user not found")
	ErrRefreshSessionNotFound = errors.New("refresh session not found")
	ErrRefreshSessionRevoked  = errors.New("refresh session revoked")
	ErrRefreshSessionExpired  = errors.New("refresh session expired")

	// service error
	ErrInvalidUser        = errors.New("invalid user")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("error unique")

	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)
