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

// VitalEvents is what this module needs from the lifeevents module: a
// person's birth and death are their birth and death life events, written
// through the person in the same transaction.
type VitalEvents interface {
	SetVitalEvent(ctx context.Context, caseID, personID, eventType string, date platform.NullQualifiedDate, place platform.NullString) error
}

type Service struct {
	store  Store
	db     *sql.DB
	log    *activitylog.Service
	rec    *activitylog.Recorder
	cases  CaseChecker
	events VitalEvents
}

func NewService(store Store, db *sql.DB, log *activitylog.Service, cases CaseChecker) *Service {
	return &Service{store: store, db: db, log: log, rec: log.For("person", store.LogName), cases: cases}
}

// SetVitalEvents supplies the lifeevents module. It is set after construction
// because the lifeevents module also depends on this one.
func (s *Service) SetVitalEvents(events VitalEvents) {
	s.events = events
}

// setVitalEvents writes the person's birth and death through their life events.
func (s *Service) setVitalEvents(ctx context.Context, caseID, personID string, input UpdatePersonInput) error {
	if s.events == nil {
		return errors.New("people: vital events not set")
	}
	if err := s.events.SetVitalEvent(ctx, caseID, personID, "birth", input.BirthDate, input.BirthPlace); err != nil {
		return err
	}
	return s.events.SetVitalEvent(ctx, caseID, personID, "death", input.DeathDate, input.DeathPlace)
}

// LockPeople and BumpVersions let the lifeevents module record a change to a
// person whose birth or death event changes (domain-model.md, rule 7). The
// person is locked before the event, the order a person's own save uses.
func (s *Service) LockPeople(ctx context.Context, ids ...string) error {
	return s.store.LockPeople(ctx, ids...)
}

func (s *Service) BumpVersions(ctx context.Context, caseID string, ids []string) error {
	return s.store.BumpVersions(ctx, caseID, ids)
}

func (s *Service) ListPeople(ctx context.Context, caseID string) ([]Person, error) {
	if err := s.cases.RequireActive(ctx, caseID); err != nil {
		return nil, platform.MapNotFound(err, ErrNotFound)
	}
	return s.store.ListPeople(ctx, caseID)
}

// DeletionImpact reports what deleting a person would take with them. For an
// active person, that is what moves to the trash with them; for a person in the
// trash, everything a permanent delete would remove. The check and the counts
// run in one transaction, so a trash or restore cannot fall between them.
func (s *Service) DeletionImpact(ctx context.Context, caseID, personID string) (*DeletionImpact, error) {
	var d *DeletionImpact
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		err := s.store.RequireActive(ctx, caseID, personID)
		switch {
		case errors.Is(err, ErrNotFound):
			d, err = s.store.TrashedDeletionImpact(ctx, caseID, personID)
		case err == nil:
			d, err = s.store.DeletionImpact(ctx, caseID, personID)
		}
		return err
	})
	return d, err
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
		if err := s.rec.Record(ctx, caseID, "created", p.ID, nil); err != nil {
			return err
		}
		if err := s.setVitalEvents(ctx, caseID, p.ID, input); err != nil {
			return err
		}
		if err := s.setLinks(ctx, caseID, p.ID, input.ParentIDs, input.ChildIDs); err != nil {
			return err
		}
		p, err = s.store.Get(ctx, caseID, p.ID)
		return err
	})
	return p, err
}

func (s *Service) UpdatePerson(ctx context.Context, caseID, personID string, input UpdatePersonInput) (*Person, error) {
	var p *Person
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID, personID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID, personID)
		if err != nil {
			return err
		}
		if err := platform.CheckVersion(before.Version, input.Version); err != nil {
			return err
		}
		if _, err = s.store.UpdatePerson(ctx, caseID, personID, input); err != nil {
			return err
		}
		if err := s.setVitalEvents(ctx, caseID, personID, input); err != nil {
			return err
		}
		if err := s.setLinks(ctx, caseID, personID, input.ParentIDs, input.ChildIDs); err != nil {
			return err
		}
		if p, err = s.store.Get(ctx, caseID, personID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "updated", personID, activitylog.Diff(before.logFields(), p.logFields()))
	})
	return p, err
}

func (s *Service) DeletePerson(ctx context.Context, caseID, personID string, version int) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID, personID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID, personID)
		if err != nil {
			return err
		}
		if err := platform.CheckVersion(before.Version, version); err != nil {
			return err
		}
		if err := s.store.DeletePerson(ctx, caseID, personID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "deleted", personID, nil)
	})
}

// setLinks makes the person's active parents and children exactly the lists
// given. A nil list leaves that side unchanged. Removals happen before
// additions, so replacing a parent stays within the two-parent limit, and every
// addition goes through the loop and limit checks.
func (s *Service) setLinks(ctx context.Context, caseID, personID string, parentIDs, childIDs *[]string) error {
	if parentIDs == nil && childIDs == nil {
		return nil
	}
	currentParents, err := s.store.ActiveParentIDs(ctx, personID)
	if err != nil {
		return err
	}
	currentChildren, err := s.store.ActiveChildIDs(ctx, personID)
	if err != nil {
		return err
	}

	// Lock everyone involved, in one ordered call, so concurrent link changes
	// touching any of them run one at a time.
	involved := append([]string{personID}, currentParents...)
	involved = append(involved, currentChildren...)
	if parentIDs != nil {
		involved = append(involved, *parentIDs...)
	}
	if childIDs != nil {
		involved = append(involved, *childIDs...)
	}
	if err := s.store.LockPeople(ctx, involved...); err != nil {
		return err
	}

	type link struct{ child, parent string }
	var remove, add []link
	if parentIDs != nil {
		for _, id := range difference(currentParents, *parentIDs) {
			remove = append(remove, link{personID, id})
		}
		for _, id := range difference(*parentIDs, currentParents) {
			add = append(add, link{personID, id})
		}
	}
	if childIDs != nil {
		for _, id := range difference(currentChildren, *childIDs) {
			remove = append(remove, link{id, personID})
		}
		for _, id := range difference(*childIDs, currentChildren) {
			add = append(add, link{id, personID})
		}
	}

	for _, l := range remove {
		if err := s.store.RemoveParent(ctx, caseID, l.child, l.parent); err != nil {
			return err
		}
		if err := s.recordLink(ctx, caseID, "deleted", l.child, l.parent); err != nil {
			return err
		}
	}
	for _, l := range add {
		if l.child == l.parent {
			return platform.Invalid("A person cannot be their own parent.")
		}
		for _, id := range []string{l.child, l.parent} {
			if err := s.store.RequireActive(ctx, caseID, id); err != nil {
				return err
			}
		}
		if _, err := s.store.AddParent(ctx, caseID, l.child, l.parent); err != nil {
			return err
		}
		if err := s.recordLink(ctx, caseID, "created", l.child, l.parent); err != nil {
			return err
		}
	}

	// A link belongs to both people it joins, so each person on the other side
	// of a changed link has changed too (domain model, rule 7). The person
	// being edited has their own version raised by the edit itself.
	others := map[string]bool{}
	for _, l := range append(remove, add...) {
		for _, id := range []string{l.child, l.parent} {
			if id != personID {
				others[id] = true
			}
		}
	}
	ids := make([]string, 0, len(others))
	for id := range others {
		ids = append(ids, id)
	}
	return s.store.BumpVersions(ctx, caseID, ids)
}

// difference returns the IDs in a that are not in b, each once.
func difference(a, b []string) []string {
	inB := make(map[string]bool, len(b))
	for _, id := range b {
		inB[id] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, id := range a {
		if !inB[id] && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
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
		if err := s.store.Restore(ctx, caseID, personID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "restored", personID, nil)
	})
}

// PermanentDelete deletes a trashed person and everything beneath them. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID, personID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		name, err := s.store.LogName(ctx, caseID, personID)
		if err != nil {
			return err
		}
		if err := s.store.PermanentDelete(ctx, caseID, personID); err != nil {
			return err
		}
		return s.rec.Named(ctx, caseID, "permanently_deleted", personID, name, nil)
	})
}

// recordLink writes an activity log entry for a parent link, logged against
// the child, named for both people.
func (s *Service) recordLink(ctx context.Context, caseID, action, childID, parentID string) error {
	child, err := s.store.LogName(ctx, caseID, childID)
	if err != nil {
		return err
	}
	parent, err := s.store.LogName(ctx, caseID, parentID)
	if err != nil {
		return err
	}
	return s.log.Insert(ctx, activitylog.InsertParams{
		CaseID: caseID, Action: action, EntityType: "person_relationship",
		EntityID: childID, EntityName: parent + " as parent of " + child,
	})
}

// LogDeletedWithPerson writes a "permanently_deleted" entry for each parent
// link the person is part of. The trash module calls it inside the permanent
// delete, before the rows are removed.
func (s *Service) LogDeletedWithPerson(ctx context.Context, caseID, personID string) error {
	links, err := s.store.LinksOf(ctx, caseID, personID)
	if err != nil {
		return err
	}
	for _, l := range links {
		if err := s.recordLink(ctx, caseID, "permanently_deleted", l.PersonID, l.ParentID); err != nil {
			return err
		}
	}
	return nil
}
