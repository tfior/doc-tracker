// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package trash

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tfior/doc-tracker/platform"
)

// ErrNotFound means the entity is not in the trash.
var ErrNotFound = fmt.Errorf("%w in trash", platform.ErrNotFound)

// CaseOwner is what this module needs from the cases module.
type CaseOwner interface {
	Restore(ctx context.Context, caseID string) error
	PermanentDelete(ctx context.Context, caseID string) error
}

// EntityOwner is what this module needs from each module that owns a
// soft-deletable entity inside a case.
type EntityOwner interface {
	Restore(ctx context.Context, caseID, id string) error
	PermanentDelete(ctx context.Context, caseID, id string) error
}

// Owners are the modules that restore and permanently delete their own records.
type Owners struct {
	Cases      CaseOwner
	People     EntityOwner
	LifeEvents EntityOwner
	Documents  EntityOwner
	ClaimLines EntityOwner
}

// Service lists what is in the trash and coordinates restore and permanent
// delete. It writes no entity tables itself: each owning module does that.
type Service struct {
	store  Store
	db     *sql.DB
	owners Owners
}

func NewService(store Store, db *sql.DB, owners Owners) *Service {
	return &Service{store: store, db: db, owners: owners}
}

func (s *Service) GetCaseTrash(ctx context.Context, caseID string) (*CaseTrash, error) {
	return s.store.GetCaseTrash(ctx, caseID)
}

func (s *Service) GetGlobalTrash(ctx context.Context) (*GlobalTrash, error) {
	return s.store.GetGlobalTrash(ctx)
}

func (s *Service) RestoreCase(ctx context.Context, caseID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.Cases.Restore(ctx, caseID) })
}

func (s *Service) PermanentDeleteCase(ctx context.Context, caseID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.Cases.PermanentDelete(ctx, caseID) })
}

func (s *Service) RestorePerson(ctx context.Context, caseID, personID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.People.Restore(ctx, caseID, personID) })
}

func (s *Service) PermanentDeletePerson(ctx context.Context, caseID, personID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.People.PermanentDelete(ctx, caseID, personID) })
}

func (s *Service) RestoreLifeEvent(ctx context.Context, caseID, eventID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.LifeEvents.Restore(ctx, caseID, eventID) })
}

func (s *Service) PermanentDeleteLifeEvent(ctx context.Context, caseID, eventID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.LifeEvents.PermanentDelete(ctx, caseID, eventID) })
}

func (s *Service) RestoreDocument(ctx context.Context, caseID, docID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.Documents.Restore(ctx, caseID, docID) })
}

func (s *Service) PermanentDeleteDocument(ctx context.Context, caseID, docID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.Documents.PermanentDelete(ctx, caseID, docID) })
}

func (s *Service) RestoreClaimLine(ctx context.Context, caseID, lineID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.ClaimLines.Restore(ctx, caseID, lineID) })
}

func (s *Service) PermanentDeleteClaimLine(ctx context.Context, caseID, lineID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.ClaimLines.PermanentDelete(ctx, caseID, lineID) })
}

// run executes one owner call as a unit of work and reports a missing entity
// as not being in the trash.
func (s *Service) run(ctx context.Context, fn func(ctx context.Context) error) error {
	return platform.MapNotFound(platform.WithTx(ctx, s.db, fn), ErrNotFound)
}
