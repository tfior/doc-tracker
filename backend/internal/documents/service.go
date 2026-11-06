// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents

import (
	"context"
	"database/sql"
	"errors"
	"time"

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
	rec        *activitylog.Recorder
	cases      CaseChecker
	people     PersonChecker
	lifeEvents LifeEventChecker
	userID     func(context.Context) (string, bool)
}

// NewService returns the documents service. userID gives the user making a
// request, who is recorded as having checked a document's details.
func NewService(store Store, db *sql.DB, log *activitylog.Service, cases CaseChecker, people PersonChecker, lifeEvents LifeEventChecker,
	userID func(context.Context) (string, bool)) *Service {
	return &Service{store: store, db: db, log: log, rec: log.For("document", store.LogName),
		cases: cases, people: people, lifeEvents: lifeEvents, userID: userID}
}

// currentUser returns the user making the request.
func (s *Service) currentUser(ctx context.Context) (string, error) {
	id, ok := s.userID(ctx)
	if !ok {
		return "", errors.New("no user in the request context")
	}
	return id, nil
}

// checkFor decides what a save does to the details check of before. Sending
// true records a new check unless the document is already checked and the
// save leaves its recorded details as they are. Changing a recorded detail
// without sending true clears the check, since it no longer covers them.
func checkFor(input UpdateDocumentInput, before *Document) checkChange {
	changed := input.changesRecordedDetails(before)
	switch {
	case input.DetailsChecked != nil && *input.DetailsChecked:
		if before.DetailsChecked && !changed {
			return checkKept
		}
		return checkSet
	case input.DetailsChecked != nil, changed && before.DetailsChecked:
		return checkCleared
	}
	return checkKept
}

func (s *Service) ListDocuments(ctx context.Context, caseID string) ([]Document, error) {
	if err := s.cases.RequireActive(ctx, caseID); err != nil {
		return nil, platform.MapNotFound(err, ErrNotFound)
	}
	docs, err := s.store.ListDocuments(ctx, caseID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range docs {
		docs[i].derive(now)
	}
	return docs, nil
}

// Progress counts each case's active Documents in the packet by readiness
// status (domain rule 38). Every case asked about is in the result.
func (s *Service) Progress(ctx context.Context, caseIDs []string) (map[string]Progress, error) {
	states, err := s.store.PacketStates(ctx, caseIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Progress, len(caseIDs))
	for _, id := range caseIDs {
		var p Progress
		for _, st := range states[id] {
			if status, ok := statusOf(st); ok {
				p.add(status)
			}
		}
		out[id] = p
	}
	return out, nil
}

func (s *Service) ListDocumentStatuses(ctx context.Context, phase string) ([]DocumentStatus, error) {
	return s.store.ListDocumentStatuses(ctx, phase)
}

// requireParents checks that the document's parent is active, and that the
// life event, if one is given, belongs to the person. A document under a life
// event depends on the event, which for a shared marriage stays active while
// either spouse is; one without an event depends on its person.
func (s *Service) requireParents(ctx context.Context, caseID, personID string, lifeEventID platform.NullString) error {
	if !lifeEventID.Set || !lifeEventID.Valid {
		return platform.MapNotFound(s.people.RequireActive(ctx, caseID, personID), ErrNotFound)
	}
	eventPerson, err := s.lifeEvents.RequireActive(ctx, caseID, lifeEventID.Value)
	if err != nil {
		return platform.MapNotFound(err, ErrNotFound)
	}
	if eventPerson != personID {
		return platform.Invalid("The life event belongs to a different person.")
	}
	return nil
}

func (s *Service) CreateDocument(ctx context.Context, caseID, personID, docType, title string, input UpdateDocumentInput) (*Document, error) {
	var d *Document
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.requireParents(ctx, caseID, personID, input.LifeEventID); err != nil {
			return err
		}
		var checkedBy string
		if input.DetailsChecked != nil && *input.DetailsChecked {
			var err error
			if checkedBy, err = s.currentUser(ctx); err != nil {
				return err
			}
		}
		var err error
		if d, err = s.store.CreateDocument(ctx, caseID, personID, docType, title, input, checkedBy); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "created", d.ID, nil)
	})
	if err != nil {
		return nil, err
	}
	d.derive(time.Now())
	return d, nil
}

// UpdateDocument changes a document in one unit of work: its details, its
// person and life event, and any of its four phase statuses. If any part is
// invalid, nothing is saved.
func (s *Service) UpdateDocument(ctx context.Context, caseID, docID string, input UpdateDocumentInput) (*Document, error) {
	var d *Document
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		currentPerson, err := s.store.LockActive(ctx, caseID, docID)
		if err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID, docID)
		if err != nil {
			return err
		}
		if err := platform.CheckVersion(before.Version, input.Version); err != nil {
			return err
		}
		if err := s.checkStatuses(ctx, input); err != nil {
			return err
		}
		if input.PersonID != nil || input.LifeEventID.Set {
			person := currentPerson
			if input.PersonID != nil {
				person = *input.PersonID
			}
			// Moving to another person without naming a life event ungroups
			// the document, so it is never left under the old person's event.
			if !input.LifeEventID.Set && person != currentPerson {
				input.LifeEventID = platform.NullString{Set: true}
			}
			if err := s.requireParents(ctx, caseID, person, input.LifeEventID); err != nil {
				return err
			}
		}
		check := checkFor(input, before)
		var userID string
		if check == checkSet {
			if userID, err = s.currentUser(ctx); err != nil {
				return err
			}
		}
		if d, err = s.store.UpdateDocument(ctx, caseID, docID, input, check, userID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "updated", docID, activitylog.Diff(before.logFields(), d.logFields()))
	})
	if err != nil {
		return nil, err
	}
	d.derive(time.Now())
	return d, nil
}

// checkStatuses reports every supplied status that does not belong to its phase.
func (s *Service) checkStatuses(ctx context.Context, input UpdateDocumentInput) error {
	var v platform.Validator
	for _, st := range []struct {
		field, phase string
		id           *string
	}{
		{"official_copy_status_id", "official_copy", input.OfficialCopy.StatusID},
		{"amendment_status_id", "amendment", input.Amendment.StatusID},
		{"apostille_status_id", "apostille", input.Apostille.StatusID},
		{"translation_status_id", "translation", input.Translation.StatusID},
	} {
		if st.id == nil {
			continue
		}
		phase, err := s.store.StatusPhase(ctx, *st.id)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err != nil || (phase != st.phase && phase != "any") {
			v.Add(st.field, "invalid_value", "This status is not available for this phase.")
		}
	}
	return v.Err()
}

func (s *Service) DeleteDocument(ctx context.Context, caseID, docID string, version int) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if _, err := s.store.LockActive(ctx, caseID, docID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID, docID)
		if err != nil {
			return err
		}
		if err := platform.CheckVersion(before.Version, version); err != nil {
			return err
		}
		if err := s.store.DeleteDocument(ctx, caseID, docID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "deleted", docID, nil)
	})
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
			if err := s.rec.Named(ctx, caseID, "updated", m.ID, m.Title, []activitylog.FieldChange{
				{Field: "person_id", From: m.FromPersonID, To: personID},
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
		// A document under a life event depends on the event, as in
		// requireParents; one without an event depends on its person.
		var parentErr error
		if lifeEventID != nil {
			_, parentErr = s.lifeEvents.RequireActive(ctx, caseID, *lifeEventID)
		} else {
			parentErr = s.people.RequireActive(ctx, caseID, personID)
		}
		if errors.Is(parentErr, platform.ErrNotFound) {
			return platform.Conflict("This document's person or life event is in the trash. Restore it first.")
		} else if parentErr != nil {
			return parentErr
		}
		if err := s.store.Restore(ctx, caseID, docID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "restored", docID, nil)
	})
}

// PermanentDelete deletes a trashed document. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID, docID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		name, err := s.store.LogName(ctx, caseID, docID)
		if err != nil {
			return err
		}
		if err := s.store.PermanentDelete(ctx, caseID, docID); err != nil {
			return err
		}
		return s.rec.Named(ctx, caseID, "permanently_deleted", docID, name, nil)
	})
}

// LogDeletedWithPerson writes a "permanently_deleted" entry for each of a person's documents. The trash
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

// LogDeletedWithLifeEvent writes a "permanently_deleted" entry for each of a life event's documents. The trash
// module calls it inside the permanent delete, before the rows are removed.
func (s *Service) LogDeletedWithLifeEvent(ctx context.Context, caseID, eventID string) error {
	ids, err := s.store.IDsForLifeEvent(ctx, caseID, eventID)
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
