// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package people

import (
	"context"
	"database/sql"
	"errors"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/platform"
)

// CaseChecker is what this module needs from the cases module.
type CaseChecker interface {
	RequireActive(ctx context.Context, caseID string) error
}

type Service struct {
	store Store
	db    *sql.DB
	log   *activitylog.Service
	cases CaseChecker
}

func NewService(store Store, db *sql.DB, log *activitylog.Service, cases CaseChecker) *Service {
	return &Service{store: store, db: db, log: log, cases: cases}
}

func (s *Service) ListPeople(ctx context.Context, caseID string, page, perPage int) ([]Person, int, error) {
	if err := s.cases.RequireActive(ctx, caseID); err != nil {
		return nil, 0, platform.MapNotFound(err, ErrNotFound)
	}
	return s.store.ListPeople(ctx, caseID, page, perPage)
}

// RequireActive returns an error wrapping platform.ErrNotFound unless the person
// exists in the case and is usable. Other modules call this through an
// interface instead of querying the people table themselves.
func (s *Service) RequireActive(ctx context.Context, caseID, personID string) error {
	return s.store.RequireActive(ctx, caseID, personID)
}

func (s *Service) CreatePerson(ctx context.Context, caseID, firstName, lastName string, input UpdatePersonInput) (*Person, error) {
	var p *Person
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.cases.RequireActive(ctx, caseID); err != nil {
			return platform.MapNotFound(err, ErrNotFound)
		}
		var err error
		if p, err = s.store.CreatePerson(ctx, caseID, firstName, lastName, input); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "created", EntityType: "person",
			EntityID: p.ID, EntityName: p.FirstName + " " + p.LastName,
		})
	})
	return p, err
}

func (s *Service) UpdatePerson(ctx context.Context, caseID, personID string, input UpdatePersonInput) (*Person, error) {
	var p *Person
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID, personID); err != nil {
			return err
		}
		var err error
		if p, err = s.store.UpdatePerson(ctx, caseID, personID, input); err != nil {
			return err
		}
		changes := activitylog.AddString(nil, "first_name", input.FirstName)
		changes = activitylog.AddString(changes, "last_name", input.LastName)
		changes = activitylog.AddNullable(changes, "birth_date", input.BirthDate)
		changes = activitylog.AddNullable(changes, "birth_place", input.BirthPlace)
		changes = activitylog.AddNullable(changes, "death_date", input.DeathDate)
		changes = activitylog.AddNullable(changes, "notes", input.Notes)
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "updated", EntityType: "person",
			EntityID: p.ID, EntityName: p.FirstName + " " + p.LastName, Changes: changes,
		})
	})
	return p, err
}

func (s *Service) DeletePerson(ctx context.Context, caseID, personID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID, personID); err != nil {
			return err
		}
		if err := s.store.DeletePerson(ctx, caseID, personID); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "deleted", EntityType: "person",
			EntityID: personID, EntityName: personID,
		})
	})
}

func (s *Service) AddParent(ctx context.Context, caseID, personID, parentID string) (*Relationship, error) {
	var rel *Relationship
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockPeople(ctx, personID, parentID); err != nil {
			return err
		}
		for _, id := range []string{personID, parentID} {
			if err := s.store.RequireActive(ctx, caseID, id); err != nil {
				return err
			}
		}
		var err error
		if rel, err = s.store.AddParent(ctx, caseID, personID, parentID); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "created", EntityType: "person_relationship",
			EntityID: personID, EntityName: "Parent relationship",
		})
	})
	return rel, err
}

func (s *Service) RemoveParent(ctx context.Context, caseID, personID, parentID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID, personID); err != nil {
			return err
		}
		if err := s.store.RemoveParent(ctx, caseID, personID, parentID); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "deleted", EntityType: "person_relationship",
			EntityID: personID, EntityName: "Parent relationship",
		})
	})
}

// Restore takes a person out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID, personID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.cases.RequireActive(ctx, caseID); errors.Is(err, platform.ErrNotFound) {
			return platform.Conflict("This person's case is in the trash. Restore the case first.")
		} else if err != nil {
			return err
		}
		child, err := s.store.ChildWithFullParents(ctx, caseID, personID)
		if err != nil {
			return err
		}
		if child != "" {
			return platform.Conflict("Restoring this person would give %s three parents. Remove one of their parent links first.", child)
		}
		return s.store.Restore(ctx, caseID, personID)
	})
}

// PermanentDelete deletes a trashed person and everything beneath them. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID, personID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.PermanentDelete(ctx, caseID, personID)
	})
}
