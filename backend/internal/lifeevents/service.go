// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package lifeevents

import (
	"context"
	"database/sql"
	"errors"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/platform"
)

// PersonChecker is what this module needs from the people module.
type PersonChecker interface {
	RequireActive(ctx context.Context, caseID, personID string) error
}

// CaseChecker is what this module needs from the cases module.
type CaseChecker interface {
	RequireActive(ctx context.Context, caseID string) error
}

// DocumentMover is what this module needs from the documents module: when a
// life event moves to another person, its documents move with it.
type DocumentMover interface {
	LockEventDocuments(ctx context.Context, caseID, eventID string) error
	MoveEventDocuments(ctx context.Context, caseID, eventID, personID string) error
}

type Service struct {
	store     Store
	db        *sql.DB
	log       *activitylog.Service
	rec       *activitylog.Recorder
	cases     CaseChecker
	people    PersonChecker
	documents DocumentMover
}

func NewService(store Store, db *sql.DB, log *activitylog.Service, cases CaseChecker, people PersonChecker) *Service {
	return &Service{store: store, db: db, log: log, rec: log.For("life_event", store.LogName), cases: cases, people: people}
}

// SetDocumentMover supplies the documents module. It is set after construction
// because the documents module also depends on this one.
func (s *Service) SetDocumentMover(documents DocumentMover) {
	s.documents = documents
}

func (s *Service) ListLifeEvents(ctx context.Context, caseID string, page, perPage int) ([]LifeEvent, int, error) {
	if err := s.cases.RequireActive(ctx, caseID); err != nil {
		return nil, 0, platform.MapNotFound(err, ErrNotFound)
	}
	return s.store.ListLifeEvents(ctx, caseID, page, perPage)
}

// RequireActive returns the event's person, or an error wrapping
// platform.ErrNotFound unless the life event is active. Other modules call
// this through an interface instead of querying the life_events table.
func (s *Service) RequireActive(ctx context.Context, caseID, eventID string) (string, error) {
	return s.store.RequireActive(ctx, caseID, eventID)
}

func (s *Service) CreateLifeEvent(ctx context.Context, caseID, personID, eventType string, input UpdateLifeEventInput) (*LifeEvent, error) {
	var le *LifeEvent
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.people.RequireActive(ctx, caseID, personID); err != nil {
			return platform.MapNotFound(err, ErrNotFound)
		}
		var err error
		if le, err = s.store.CreateLifeEvent(ctx, caseID, personID, eventType, input); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "created", le.ID, nil)
	})
	return le, err
}

func (s *Service) UpdateLifeEvent(ctx context.Context, caseID, eventID string, input UpdateLifeEventInput) (*LifeEvent, error) {
	var le *LifeEvent
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if _, err := s.store.LockActive(ctx, caseID, eventID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		if le, err = s.store.UpdateLifeEvent(ctx, caseID, eventID, input); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "updated", eventID, activitylog.Diff(before.logFields(), le.logFields()))
	})
	return le, err
}

func (s *Service) DeleteLifeEvent(ctx context.Context, caseID, eventID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if _, err := s.store.LockActive(ctx, caseID, eventID); err != nil {
			return err
		}
		if err := s.store.DeleteLifeEvent(ctx, caseID, eventID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "deleted", eventID, nil)
	})
}

func (s *Service) ReassignLifeEvent(ctx context.Context, caseID, eventID, personID string) (*LifeEvent, error) {
	var le *LifeEvent
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if s.documents == nil {
			return errors.New("lifeevents: document mover not set")
		}
		if err := s.documents.LockEventDocuments(ctx, caseID, eventID); err != nil {
			return err
		}
		if _, err := s.store.LockActive(ctx, caseID, eventID); err != nil {
			return err
		}
		if err := s.people.RequireActive(ctx, caseID, personID); err != nil {
			return platform.MapNotFound(err, ErrNotFound)
		}
		before, err := s.store.Get(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		if le, err = s.store.ReassignLifeEvent(ctx, caseID, eventID, personID); err != nil {
			return err
		}
		if err := s.documents.MoveEventDocuments(ctx, caseID, eventID, personID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "updated", eventID, activitylog.Diff(before.logFields(), le.logFields()))
	})
	return le, err
}

// Restore takes a life event out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID, eventID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		personID, err := s.store.TrashedPersonID(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		if err := s.people.RequireActive(ctx, caseID, personID); errors.Is(err, platform.ErrNotFound) {
			return platform.Conflict("This life event's person is in the trash. Restore the person first.")
		} else if err != nil {
			return err
		}
		if err := s.store.Restore(ctx, caseID, eventID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "restored", eventID, nil)
	})
}

// PermanentDelete deletes a trashed life event. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID, eventID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if s.documents == nil {
			return errors.New("lifeevents: document mover not set")
		}
		if err := s.documents.LockEventDocuments(ctx, caseID, eventID); err != nil {
			return err
		}
		name, err := s.store.LogName(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		if err := s.store.PermanentDelete(ctx, caseID, eventID); err != nil {
			return err
		}
		return s.rec.Named(ctx, caseID, "permanently_deleted", eventID, name, nil)
	})
}

// LogDeletedWithPerson writes a "permanently_deleted" entry for each of a person's life events. The trash
// module calls it inside the permanent delete, before the rows are removed.
func (s *Service) LogDeletedWithPerson(ctx context.Context, caseID, personID string) error {
	ids, err := s.store.IDsForPerson(ctx, caseID, personID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.rec.Record(ctx, caseID, "permanently_deleted", id, nil); err != nil {
			return err
		}
	}
	return nil
}
