// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package activitylog

import (
	"context"
	"errors"
)

// ErrNoUser is returned when a log entry is written without a user in the
// context. It aborts the surrounding transaction, so no unattributed change
// can be saved.
var ErrNoUser = errors.New("activity log: no authenticated user in context")

// UserFunc reports the authenticated user carried by a request context.
type UserFunc func(ctx context.Context) (string, bool)

type Service struct {
	store  Store
	userID UserFunc
}

// NewService takes the function that identifies the current user, so this
// package does not depend on the auth module.
func NewService(store Store, userID UserFunc) *Service {
	return &Service{store: store, userID: userID}
}

// Insert records an activity log entry for the current user. It must be called
// inside the transaction of the change it records: if it fails, the caller
// returns the error and the whole change is rolled back.
func (s *Service) Insert(ctx context.Context, p InsertParams) error {
	if p.UserID == "" {
		id, ok := s.userID(ctx)
		if !ok || id == "" {
			return ErrNoUser
		}
		p.UserID = id
	}
	return s.store.Insert(ctx, p)
}
