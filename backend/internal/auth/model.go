package auth

import (
	"errors"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Session struct {
	Token     string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}
