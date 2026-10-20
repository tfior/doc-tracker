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
//
// An update that changed nothing is not logged: Insert returns nil without
// writing.
func (s *Service) Insert(ctx context.Context, p InsertParams) error {
	if p.Action == "updated" && len(p.Changes) == 0 {
		return nil
	}
	if p.UserID == "" {
		id, ok := s.userID(ctx)
		if !ok || id == "" {
			return ErrNoUser
		}
		p.UserID = id
	}
	return s.store.Insert(ctx, p)
}

// NameFunc returns the current name of a record for the activity log.
type NameFunc func(ctx context.Context, caseID, id string) (string, error)

// Recorder writes activity log entries for one entity type, naming each entry
// with the module's NameFunc.
type Recorder struct {
	log        *Service
	entityType string
	name       NameFunc
}

// For returns a Recorder for an entity type.
func (s *Service) For(entityType string, name NameFunc) *Recorder {
	return &Recorder{log: s, entityType: entityType, name: name}
}

// Record writes an entry named as the record is now. An update with no
// changes is skipped before the name is looked up.
func (r *Recorder) Record(ctx context.Context, caseID, action, id string, changes []FieldChange) error {
	if action == "updated" && len(changes) == 0 {
		return nil
	}
	name, err := r.name(ctx, caseID, id)
	if err != nil {
		return err
	}
	return r.Named(ctx, caseID, action, id, name, changes)
}

// Named writes an entry with a name already looked up, for entries written
// after the record itself is gone.
func (r *Recorder) Named(ctx context.Context, caseID, action, id, name string, changes []FieldChange) error {
	return r.log.Insert(ctx, InsertParams{
		CaseID: caseID, Action: action, EntityType: r.entityType,
		EntityID: id, EntityName: name, Changes: changes,
	})
}
