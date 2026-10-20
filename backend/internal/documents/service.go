// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents

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

// LifeEventChecker is what this module needs from the lifeevents module. It
// returns the event's person.
type LifeEventChecker interface {
	RequireActive(ctx context.Context, caseID, eventID string) (personID string, err error)
}

// CaseChecker is what this module needs from the cases module.
type CaseChecker interface {
	RequireActive(ctx context.Context, caseID string) error
}

type Service struct {
	store      Store
	db         *sql.DB
	log        *activitylog.Service
	cases      CaseChecker
	people     PersonChecker
	lifeEvents LifeEventChecker
}

func NewService(store Store, db *sql.DB, log *activitylog.Service, cases CaseChecker, people PersonChecker, lifeEvents LifeEventChecker) *Service {
	return &Service{store: store, db: db, log: log, cases: cases, people: people, lifeEvents: lifeEvents}
}

func (s *Service) ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error) {
	if err := s.cases.RequireActive(ctx, caseID); err != nil {
		return nil, 0, platform.MapNotFound(err, ErrNotFound)
	}
	return s.store.ListDocuments(ctx, caseID, page, perPage)
}

func (s *Service) ListDocumentStatuses(ctx context.Context, phase string) ([]DocumentStatus, error) {
	return s.store.ListDocumentStatuses(ctx, phase)
}

// requireParents checks that the person, and the life event if one is given,
// are active, and that the life event belongs to that person.
func (s *Service) requireParents(ctx context.Context, caseID, personID string, lifeEventID platform.NullString) error {
	if err := s.people.RequireActive(ctx, caseID, personID); err != nil {
		return platform.MapNotFound(err, ErrNotFound)
	}
	if lifeEventID.Set && lifeEventID.Valid {
		eventPerson, err := s.lifeEvents.RequireActive(ctx, caseID, lifeEventID.Value)
		if err != nil {
			return platform.MapNotFound(err, ErrNotFound)
		}
		if eventPerson != personID {
			return platform.Invalid("The life event belongs to a different person.")
		}
	}
	return nil
}

func (s *Service) CreateDocument(ctx context.Context, caseID, personID, docType, title string, input UpdateDocumentInput) (*Document, error) {
	var d *Document
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.requireParents(ctx, caseID, personID, input.LifeEventID); err != nil {
			return err
		}
		var err error
		if d, err = s.store.CreateDocument(ctx, caseID, personID, docType, title, input); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "created", EntityType: "document",
			EntityID: d.ID, EntityName: d.Title,
		})
	})
	return d, err
}

func (s *Service) UpdateDocument(ctx context.Context, caseID, docID string, input UpdateDocumentInput) (*Document, error) {
	var d *Document
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if _, err := s.store.LockActive(ctx, caseID, docID); err != nil {
			return err
		}
		var err error
		if d, err = s.store.UpdateDocument(ctx, caseID, docID, input); err != nil {
			return err
		}
		changes := activitylog.AddString(nil, "title", input.Title)
		changes = activitylog.AddString(changes, "document_type", input.DocumentType)
		changes = activitylog.AddBool(changes, "is_verified", input.IsVerified)
		changes = activitylog.AddNullable(changes, "issuing_authority", input.IssuingAuthority)
		changes = activitylog.AddNullable(changes, "issue_date", input.IssueDate)
		changes = activitylog.AddNullable(changes, "recorded_date", input.RecordedDate)
		changes = activitylog.AddNullable(changes, "recorded_given_name", input.RecordedGivenName)
		changes = activitylog.AddNullable(changes, "recorded_surname", input.RecordedSurname)
		changes = activitylog.AddNullable(changes, "recorded_birth_date", input.RecordedBirthDate)
		changes = activitylog.AddNullable(changes, "recorded_birth_place", input.RecordedBirthPlace)
		changes = activitylog.AddNullable(changes, "notes", input.Notes)
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "updated", EntityType: "document",
			EntityID: d.ID, EntityName: d.Title, Changes: changes,
		})
	})
	return d, err
}

func (s *Service) DeleteDocument(ctx context.Context, caseID, docID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if _, err := s.store.LockActive(ctx, caseID, docID); err != nil {
			return err
		}
		if err := s.store.DeleteDocument(ctx, caseID, docID); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "deleted", EntityType: "document",
			EntityID: docID, EntityName: docID,
		})
	})
}

func (s *Service) TransitionStatus(ctx context.Context, caseID, docID string, input TransitionStatusInput) (*Document, error) {
	var d *Document
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if _, err := s.store.LockActive(ctx, caseID, docID); err != nil {
			return err
		}
		var err error
		if d, err = s.store.TransitionStatus(ctx, caseID, docID, input); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "updated", EntityType: "document",
			EntityID: d.ID, EntityName: d.Title,
			Changes: []activitylog.FieldChange{{Field: input.Phase + "_status", To: input.StatusID}},
		})
	})
	return d, err
}

func (s *Service) ReassignDocument(ctx context.Context, caseID, docID string, input ReassignDocumentInput) (*Document, error) {
	var d *Document
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		currentPerson, err := s.store.LockActive(ctx, caseID, docID)
		if err != nil {
			return err
		}
		if err := s.requireParents(ctx, caseID, input.PersonID, input.LifeEventID); err != nil {
			return err
		}
		// Moving to another person without naming a life event ungroups the
		// document, so it is never left under the old person's event.
		if !input.LifeEventID.Set && input.PersonID != currentPerson {
			input.LifeEventID = platform.NullString{Set: true}
		}
		if d, err = s.store.ReassignDocument(ctx, caseID, docID, input); err != nil {
			return err
		}
		changes := []activitylog.FieldChange{{Field: "person_id", To: input.PersonID}}
		changes = activitylog.AddNullable(changes, "life_event_id", input.LifeEventID)
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "updated", EntityType: "document",
			EntityID: d.ID, EntityName: d.Title, Changes: changes,
		})
	})
	return d, err
}

// LockEventDocuments locks every document under a life event. The lifeevents
// module calls it before locking the event itself.
func (s *Service) LockEventDocuments(ctx context.Context, caseID, eventID string) error {
	return s.store.LockEventDocuments(ctx, caseID, eventID)
}

// MoveEventDocuments gives every document under a life event the event's new
// person. Called by the lifeevents module inside its own unit of work.
func (s *Service) MoveEventDocuments(ctx context.Context, caseID, eventID, personID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		moved, err := s.store.MoveEventDocuments(ctx, caseID, eventID, personID)
		if err != nil {
			return err
		}
		for _, m := range moved {
			if err := s.log.Insert(ctx, activitylog.InsertParams{
				CaseID: caseID, Action: "updated", EntityType: "document",
				EntityID: m.ID, EntityName: m.Title,
				Changes: []activitylog.FieldChange{{Field: "person_id", To: personID}},
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// Restore takes a document out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID, docID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		personID, lifeEventID, err := s.store.TrashedParents(ctx, caseID, docID)
		if err != nil {
			return err
		}
		parentTrashed := platform.Conflict("This document's person or life event is in the trash. Restore it first.")
		if err := s.people.RequireActive(ctx, caseID, personID); errors.Is(err, platform.ErrNotFound) {
			return parentTrashed
		} else if err != nil {
			return err
		}
		if lifeEventID != nil {
			if _, err := s.lifeEvents.RequireActive(ctx, caseID, *lifeEventID); errors.Is(err, platform.ErrNotFound) {
				return parentTrashed
			} else if err != nil {
				return err
			}
		}
		return s.store.Restore(ctx, caseID, docID)
	})
}

// PermanentDelete deletes a trashed document. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID, docID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.PermanentDelete(ctx, caseID, docID)
	})
}
