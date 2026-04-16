package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/tfior/doc-tracker/internal/users"
)

const sessionTTL = 24 * time.Hour

type Service struct {
	store    *SessionStore
	usersSvc *users.Service
}

func NewService(store *SessionStore, usersSvc *users.Service) *Service {
	return &Service{store: store, usersSvc: usersSvc}
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.usersSvc.GetByEmail(ctx, email)
	if errors.Is(err, users.ErrNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if !s.usersSvc.CheckPassword(user.PasswordHash, password) {
		return "", ErrInvalidCredentials
	}
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	s.store.Create(token, user.ID, sessionTTL)
	return token, nil
}

func (s *Service) Logout(token string) {
	s.store.Delete(token)
}

func (s *Service) GetSession(token string) (Session, bool) {
	return s.store.Get(token)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
