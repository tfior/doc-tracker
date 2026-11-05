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

// PersonChecker is what this module needs from the people module. A person's
// birth and death events are part of the person, so a change to either is
// recorded as a change to the person too (domain-model.md, rule 7). The person
// is locked before the event, the order a person's own save uses.
type PersonChecker interface {
	RequireActive(ctx context.Context, caseID, personID string) error
	LockPeople(ctx context.Context, ids ...string) error
	BumpVersions(ctx context.Context, caseID string, ids []string) error
}

// isVital reports whether an event type is one a person has at most one of,
// and which is part of the person: birth or death.
func isVital(eventType string) bool {
	return eventType == "birth" || eventType == "death"
}

// vitalPeople lists, once each, the people whose birth or death event a change
// touches: the person it belonged to and the person it belongs to after.
func vitalPeople(beforePerson, beforeType, afterPerson, afterType string) []string {
	var ids []string
	if isVital(beforeType) {
		ids = append(ids, beforePerson)
	}
	if isVital(afterType) && (afterPerson != beforePerson || !isVital(beforeType)) {
		ids = append(ids, afterPerson)
	}
	return ids
}

// lockPeople locks the people a change touches, before anything else, so the
// change can be recorded against them without risk of deadlock.
func (s *Service) lockPeople(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.people.LockPeople(ctx, ids...)
}

// owner is the person a life event belongs to, and its linked spouse, as read
// before the event is locked, and the people a change to it touches.
type owner struct {
	Peeked
	touched []string
}

// lockOwner reads which person a life event belongs to and, when it is their
// birth or death event, locks them, for a change that keeps the event with
// its person: a delete or a restore.
func (s *Service) lockOwner(ctx context.Context, caseID, eventID string) (owner, error) {
	var o owner
	var err error
	if o.Peeked, err = s.store.Peek(ctx, caseID, eventID); err != nil {
		return o, err
	}
	o.touched = vitalPeople("", "", o.PersonID, o.EventType)
	return o, s.lockPeople(ctx, o.touched)
}

// refuseSecondVital refuses a write that would give a person a second birth or
// death event (domain-model.md, rule 6).
func (s *Service) refuseSecondVital(ctx context.Context, caseID, personID, eventType, exceptID, refusal string) error {
	if !isVital(eventType) {
		return nil
	}
	name, err := s.store.VitalHolder(ctx, caseID, personID, eventType, exceptID)
	if err != nil || name == "" {
		return err
	}
	return platform.Conflict(refusal, name, eventType)
}

// Refusals of spouse and divorce details (domain-model.md, rules 22 and 23).
const (
	notMarriage   = "Only a marriage can have spouse or divorce details."
	linkedSpouse  = "A linked spouse's details come from their person record. Clear this field to link them."
	ownSpouse     = "A person cannot be their own spouse."
	spouseMissing = "Choose a person in this case who is not in the trash."
)

// requireSpouse checks, before the event is locked, that a spouse being linked
// is an active person in the case. It returns platform.ErrNotFound when they
// are not, for checkMarriage to report.
func (s *Service) requireSpouse(ctx context.Context, caseID string, input UpdateLifeEventInput) error {
	if !input.SpousePersonID.Valid {
		return nil
	}
	return s.people.RequireActive(ctx, caseID, input.SpousePersonID.Value)
}

// checkMarriage checks the spouse and divorce details a write would leave on
// an event: only a marriage has them; a linked spouse is another active
// person in the case; and a marriage with a linked spouse has no flat spouse
// details. before is the event as stored, or nil on create; spouseErr is what
// requireSpouse returned.
func checkMarriage(before *LifeEvent, personID, eventType string, input UpdateLifeEventInput, spouseErr error) error {
	if spouseErr != nil && !errors.Is(spouseErr, platform.ErrNotFound) {
		return spouseErr
	}
	if before == nil {
		before = &LifeEvent{}
	}
	// has reports whether a field holds a value after the write.
	has := func(stored bool, set, valid bool) bool {
		if set {
			return valid
		}
		return stored
	}
	spouse := ""
	if before.SpousePersonID != nil {
		spouse = *before.SpousePersonID
	}
	if input.SpousePersonID.Set {
		spouse = input.SpousePersonID.Value
	}
	flat := []struct {
		field string
		has   bool
	}{
		{"spouse_name", has(before.SpouseName != nil, input.SpouseName.Set, input.SpouseName.Valid)},
		{"spouse_birth_date", has(before.SpouseBirthDate != nil, input.SpouseBirthDate.Set, input.SpouseBirthDate.Valid)},
		{"spouse_birth_place", has(before.SpouseBirthPlace != nil, input.SpouseBirthPlace.Set, input.SpouseBirthPlace.Valid)},
	}

	var v platform.Validator
	if eventType != "marriage" {
		if spouse != "" {
			v.Add("spouse_person_id", "invalid_value", notMarriage)
		}
		for _, f := range flat {
			if f.has {
				v.Add(f.field, "invalid_value", notMarriage)
			}
		}
		if has(before.DivorceDate != nil, input.DivorceDate.Set, input.DivorceDate.Valid) {
			v.Add("divorce_date", "invalid_value", notMarriage)
		}
		if has(before.DivorcePlace != nil, input.DivorcePlace.Set, input.DivorcePlace.Valid) {
			v.Add("divorce_place", "invalid_value", notMarriage)
		}
		return v.Err()
	}
	if spouse != "" {
		for _, f := range flat {
			if f.has {
				v.Add(f.field, "invalid_value", linkedSpouse)
			}
		}
	}
	switch {
	case spouse != "" && spouse == personID && input.SpousePersonID.Set:
		v.Add("spouse_person_id", "invalid_value", ownSpouse)
	case spouse != "" && spouse == personID:
		v.Add("person_id", "invalid_value", "A marriage cannot be moved to its own spouse.")
	case spouseErr != nil:
		v.Add("spouse_person_id", "invalid_value", spouseMissing)
	}
	return v.Err()
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

func (s *Service) ListLifeEvents(ctx context.Context, caseID string) ([]LifeEvent, error) {
	if err := s.cases.RequireActive(ctx, caseID); err != nil {
		return nil, platform.MapNotFound(err, ErrNotFound)
	}
	return s.store.ListLifeEvents(ctx, caseID)
}

// DeletionImpact reports what deleting a life event would take with it. For an
// active event, that is what moves to the trash with it; for an event in the
// trash, everything a permanent delete would remove. The check and the counts
// run in one transaction, so a trash or restore cannot fall between them.
func (s *Service) DeletionImpact(ctx context.Context, caseID, eventID string) (*DeletionImpact, error) {
	var d *DeletionImpact
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		_, err := s.store.RequireActive(ctx, caseID, eventID)
		switch {
		case errors.Is(err, ErrNotFound):
			d, err = s.store.TrashedDeletionImpact(ctx, caseID, eventID)
		case err == nil:
			d, err = s.store.DeletionImpact(ctx, caseID, eventID)
		}
		return err
	})
	return d, err
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
		touched := vitalPeople("", "", personID, eventType)
		if err := s.lockPeople(ctx, touched); err != nil {
			return err
		}
		if err := s.people.RequireActive(ctx, caseID, personID); err != nil {
			return platform.MapNotFound(err, ErrNotFound)
		}
		if err := checkMarriage(nil, personID, eventType, input, s.requireSpouse(ctx, caseID, input)); err != nil {
			return err
		}
		if err := s.refuseSecondVital(ctx, caseID, personID, eventType, "", "%s already has a %s event."); err != nil {
			return err
		}
		var err error
		if le, err = s.store.CreateLifeEvent(ctx, caseID, personID, eventType, input); err != nil {
			return err
		}
		if err := s.people.BumpVersions(ctx, caseID, touched); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "created", le.ID, nil)
	})
	return le, err
}

// SetVitalEvent writes a person's birth or death date and place to their
// birth or death event: the existing one, or a new one when a value is given
// and they have none. Clearing both leaves the event, which may hold documents.
// Called by the people module inside a person's save, which has already
// locked the person and records the change to them itself.
func (s *Service) SetVitalEvent(ctx context.Context, caseID, personID, eventType string, date platform.NullQualifiedDate, place platform.NullString) error {
	if !date.Set && !place.Set {
		return nil
	}
	input := UpdateLifeEventInput{EventDate: date, EventPlace: place}
	id, err := s.store.VitalEvent(ctx, caseID, personID, eventType)
	if err != nil {
		return err
	}
	if id == "" {
		if !date.Valid && !place.Valid {
			return nil
		}
		le, err := s.store.CreateLifeEvent(ctx, caseID, personID, eventType, input)
		if err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "created", le.ID, nil)
	}
	before, err := s.store.Get(ctx, caseID, id)
	if err != nil {
		return err
	}
	after, err := s.store.UpdateLifeEvent(ctx, caseID, id, input)
	if err != nil {
		return err
	}
	return s.rec.Record(ctx, caseID, "updated", id, activitylog.Diff(before.logFields(), after.logFields()))
}

// UpdateLifeEvent changes a life event's details and, when PersonID is given,
// moves it and its documents to another person.
func (s *Service) UpdateLifeEvent(ctx context.Context, caseID, eventID string, input UpdateLifeEventInput) (*LifeEvent, error) {
	var le *LifeEvent
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if input.PersonID != nil {
			if s.documents == nil {
				return errors.New("lifeevents: document mover not set")
			}
			// Documents are locked before their event, the order every
			// document write uses, so the two cannot deadlock.
			if err := s.documents.LockEventDocuments(ctx, caseID, eventID); err != nil {
				return err
			}
		}
		// The people the change touches are locked next, before the event.
		// What the event was is read unlocked to find them; if it changes
		// before the event is locked, the version check below refuses the write.
		peek, err := s.store.Peek(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		peekPerson, peekType := peek.PersonID, peek.EventType
		afterPerson, afterType := peekPerson, peekType
		if input.PersonID != nil {
			afterPerson = *input.PersonID
		}
		if input.EventType != nil {
			afterType = *input.EventType
		}
		touched := vitalPeople(peekPerson, peekType, afterPerson, afterType)
		if err := s.lockPeople(ctx, touched); err != nil {
			return err
		}
		spouseErr := s.requireSpouse(ctx, caseID, input)
		if _, err := s.store.LockActive(ctx, caseID, eventID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		if err := platform.CheckVersion(before.Version, input.Version); err != nil {
			return err
		}
		if err := checkMarriage(before, afterPerson, afterType, input, spouseErr); err != nil {
			return err
		}
		moving := input.PersonID != nil && *input.PersonID != before.PersonID
		if moving {
			if err := s.people.RequireActive(ctx, caseID, *input.PersonID); err != nil {
				return platform.MapNotFound(err, ErrNotFound)
			}
		}
		if afterType != before.EventType || afterPerson != before.PersonID {
			if err := s.refuseSecondVital(ctx, caseID, afterPerson, afterType, eventID, "%s already has a %s event."); err != nil {
				return err
			}
		}
		if le, err = s.store.UpdateLifeEvent(ctx, caseID, eventID, input); err != nil {
			return err
		}
		if err := s.people.BumpVersions(ctx, caseID, touched); err != nil {
			return err
		}
		if moving {
			if err := s.documents.MoveEventDocuments(ctx, caseID, eventID, *input.PersonID); err != nil {
				return err
			}
		}
		return s.rec.Record(ctx, caseID, "updated", eventID, activitylog.Diff(before.logFields(), le.logFields()))
	})
	return le, err
}

func (s *Service) DeleteLifeEvent(ctx context.Context, caseID, eventID string, version int) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		o, err := s.lockOwner(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		if _, err := s.store.LockActive(ctx, caseID, eventID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		if err := platform.CheckVersion(before.Version, version); err != nil {
			return err
		}
		if err := s.store.DeleteLifeEvent(ctx, caseID, eventID); err != nil {
			return err
		}
		if err := s.people.BumpVersions(ctx, caseID, o.touched); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "deleted", eventID, nil)
	})
}

// Restore takes a life event out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID, eventID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		o, err := s.lockOwner(ctx, caseID, eventID)
		if err != nil {
			return err
		}
		if !o.Trashed {
			return ErrNotFound
		}
		if err := s.requireEitherSpouse(ctx, caseID, o.Peeked); err != nil {
			return err
		}
		if err := s.refuseSecondVital(ctx, caseID, o.PersonID, o.EventType, eventID,
			"%s already has another %s event. Delete it or change its type before restoring this one."); err != nil {
			return err
		}
		if err := s.store.Restore(ctx, caseID, eventID); err != nil {
			return err
		}
		if err := s.people.BumpVersions(ctx, caseID, o.touched); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "restored", eventID, nil)
	})
}

// requireEitherSpouse refuses to restore a life event unless one of its people
// is active: its person, or for a shared marriage either spouse (rule 26).
func (s *Service) requireEitherSpouse(ctx context.Context, caseID string, p Peeked) error {
	err := s.people.RequireActive(ctx, caseID, p.PersonID)
	if errors.Is(err, platform.ErrNotFound) && p.SpouseID != "" {
		if err = s.people.RequireActive(ctx, caseID, p.SpouseID); errors.Is(err, platform.ErrNotFound) {
			return platform.Conflict("Both people in this marriage are in the trash. Restore one of them first.")
		}
	}
	if errors.Is(err, platform.ErrNotFound) {
		return platform.Conflict("This life event's person is in the trash. Restore the person first.")
	}
	return err
}

// HandOverMarriages keeps a person's shared marriages, and their documents,
// when the person is permanently deleted (rule 26). Each passes to the other
// spouse, unlinked, with the departing spouse's name, birth date and birth
// place kept in the flat spouse fields. Called by the trash module inside the
// permanent delete, before anything is logged or removed.
func (s *Service) HandOverMarriages(ctx context.Context, caseID, personID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if s.documents == nil {
			return errors.New("lifeevents: document mover not set")
		}
		// Locks follow the order every write takes them: the documents that
		// move, then the departing person, then the marriages. The marriages
		// are read unlocked to find the documents, then again once locked.
		peeked, err := s.store.SharedMarriages(ctx, caseID, personID, false)
		if err != nil || len(peeked) == 0 {
			return err
		}
		docsLocked := map[string]bool{}
		lockDocs := func(m SharedMarriage) error {
			if m.PersonID != personID || docsLocked[m.ID] {
				return nil
			}
			docsLocked[m.ID] = true
			return s.documents.LockEventDocuments(ctx, caseID, m.ID)
		}
		for _, m := range peeked {
			if err := lockDocs(m); err != nil {
				return err
			}
		}
		if err := s.people.LockPeople(ctx, personID); err != nil {
			return err
		}
		marriages, err := s.store.SharedMarriages(ctx, caseID, personID, true)
		if err != nil {
			return err
		}
		details, err := s.store.SpouseDetails(ctx, caseID, personID)
		if err != nil {
			return err
		}
		for _, m := range marriages {
			survivor := m.PersonID
			if survivor == personID {
				survivor = m.SpouseID
				if err := lockDocs(m); err != nil {
					return err
				}
			}
			before, after, err := s.store.HandOver(ctx, caseID, m.ID, survivor, details)
			if err != nil {
				return err
			}
			if survivor != m.PersonID {
				if err := s.documents.MoveEventDocuments(ctx, caseID, m.ID, survivor); err != nil {
					return err
				}
			}
			if err := s.rec.Record(ctx, caseID, "updated", m.ID, activitylog.Diff(before.logFields(), after.logFields())); err != nil {
				return err
			}
		}
		return nil
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
		// An event in the trash is no longer part of its person, so deleting
		// it for good is not a change to them.
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
