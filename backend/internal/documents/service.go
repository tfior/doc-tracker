// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents

import (
	"context"
	"database/sql"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/platform"
)

// PersonChecker is what this module needs from the people module.
type PersonChecker interface {
	RequireActive(ctx context.Context, caseID, personID string) error
}

// LifeEventChecker is what this module needs from the lifeevents module.
type LifeEventChecker interface {
	RequireActive(ctx context.Context, caseID, eventID string) error
}

type Service struct {
	store      Store
	db         *sql.DB
	log        *activitylog.Service
	people     PersonChecker
	lifeEvents LifeEventChecker
}

func NewService(store Store, db *sql.DB, log *activitylog.Service, people PersonChecker, lifeEvents LifeEventChecker) *Service {
	return &Service{store: store, db: db, log: log, people: people, lifeEvents: lifeEvents}
}

func (s *Service) ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error) {
	return s.store.ListDocuments(ctx, caseID, page, perPage)
}

func (s *Service) ListDocumentStatuses(ctx context.Context, phase string) ([]DocumentStatus, error) {
	return s.store.ListDocumentStatuses(ctx, phase)
}

// requireParents checks that the person, and the life event if one is given,
// exist in the case and are usable.
func (s *Service) requireParents(ctx context.Context, caseID, personID string, lifeEventID platform.NullString) error {
	if err := s.people.RequireActive(ctx, caseID, personID); err != nil {
		return platform.MapNotFound(err, ErrNotFound)
	}
	if lifeEventID.Set && lifeEventID.Valid {
		if err := s.lifeEvents.RequireActive(ctx, caseID, lifeEventID.Value); err != nil {
			return platform.MapNotFound(err, ErrNotFound)
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
		if err := s.requireParents(ctx, caseID, input.PersonID, input.LifeEventID); err != nil {
			return err
		}
		var err error
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

// Restore takes a document out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID, docID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.Restore(ctx, caseID, docID)
	})
}

// PermanentDelete deletes a trashed document. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID, docID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.PermanentDelete(ctx, caseID, docID)
	})
}
