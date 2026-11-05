// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package users

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetByEmail(ctx, email)
}

func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, email, firstName, lastName, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.store.Create(ctx, email, firstName, lastName, string(hash))
}

// SetPassword replaces a user's password.
func (s *Service) SetPassword(ctx context.Context, id, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.SetPasswordHash(ctx, id, string(hash))
}

// LockedPasswordHash is stored in place of a password hash to lock an account.
// It is not a well-formed bcrypt hash, so CheckPassword rejects every password
// against it, including "!" itself.
const LockedPasswordHash = "!"

// Lock stops any new login to an account, until a password is set again.
// Sessions already open are not ended.
func (s *Service) Lock(ctx context.Context, id string) error {
	return s.store.SetPasswordHash(ctx, id, LockedPasswordHash)
}

func (s *Service) CheckPassword(passwordHash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}
