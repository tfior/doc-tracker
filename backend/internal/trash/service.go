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

// PersonDependent is a module whose records are permanently deleted along with
// a person. It logs them before they go.
type PersonDependent interface {
	LogDeletedWithPerson(ctx context.Context, caseID, personID string) error
}

// LifeEventDependent is a module whose records are permanently deleted along
// with a life event. It logs them before they go.
type LifeEventDependent interface {
	LogDeletedWithLifeEvent(ctx context.Context, caseID, eventID string) error
}

// Owners are the modules that restore and permanently delete their own records,
// and the modules whose records go with them.
type Owners struct {
	Cases      CaseOwner
	People     EntityOwner
	LifeEvents EntityOwner
	Documents  EntityOwner
	ClaimLines EntityOwner

	WithPerson    []PersonDependent
	WithLifeEvent []LifeEventDependent
}

// UserFunc reports the authenticated user carried by a request context.
type UserFunc func(ctx context.Context) (string, bool)

// Service lists what is in the trash and coordinates restore and permanent
// delete. It writes no entity tables itself: each owning module does that.
type Service struct {
	store  Store
	db     *sql.DB
	owners Owners
	userID UserFunc
}

func NewService(store Store, db *sql.DB, owners Owners, userID UserFunc) *Service {
	return &Service{store: store, db: db, owners: owners, userID: userID}
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

// PermanentDeleteCase deletes a trashed case and writes its deletion record in
// the same transaction: if the record cannot be written, the case is kept.
func (s *Service) PermanentDeleteCase(ctx context.Context, caseID string) error {
	return s.run(ctx, func(ctx context.Context) error {
		record, err := s.store.CaseDeletionRecord(ctx, caseID)
		if err != nil {
			return err
		}
		if err := s.owners.Cases.PermanentDelete(ctx, caseID); err != nil {
			return err
		}
		record.DeletedBy, _ = s.userID(ctx)
		return s.store.InsertCaseDeletionRecord(ctx, record)
	})
}

func (s *Service) RestorePerson(ctx context.Context, caseID, personID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.People.Restore(ctx, caseID, personID) })
}

// PermanentDeletePerson deletes a trashed person, and logs every record that
// goes with them, in one transaction.
func (s *Service) PermanentDeletePerson(ctx context.Context, caseID, personID string) error {
	return s.run(ctx, func(ctx context.Context) error {
		for _, d := range s.owners.WithPerson {
			if err := d.LogDeletedWithPerson(ctx, caseID, personID); err != nil {
				return err
			}
		}
		return s.owners.People.PermanentDelete(ctx, caseID, personID)
	})
}

func (s *Service) RestoreLifeEvent(ctx context.Context, caseID, eventID string) error {
	return s.run(ctx, func(ctx context.Context) error { return s.owners.LifeEvents.Restore(ctx, caseID, eventID) })
}

// PermanentDeleteLifeEvent deletes a trashed life event, and logs every
// document that goes with it, in one transaction.
func (s *Service) PermanentDeleteLifeEvent(ctx context.Context, caseID, eventID string) error {
	return s.run(ctx, func(ctx context.Context) error {
		for _, d := range s.owners.WithLifeEvent {
			if err := d.LogDeletedWithLifeEvent(ctx, caseID, eventID); err != nil {
				return err
			}
		}
		return s.owners.LifeEvents.PermanentDelete(ctx, caseID, eventID)
	})
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
