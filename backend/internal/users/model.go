package users

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("user not found")
var ErrEmailTaken = errors.New("email already in use")

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
