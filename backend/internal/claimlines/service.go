// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package claimlines

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

type Service struct {
	store  Store
	db     *sql.DB
	log    *activitylog.Service
	rec    *activitylog.Recorder
	cases  CaseChecker
	people PersonChecker
}

func NewService(store Store, db *sql.DB, log *activitylog.Service, cases CaseChecker, people PersonChecker) *Service {
	return &Service{store: store, db: db, log: log, rec: log.For("claim_line", store.LogName), cases: cases, people: people}
}

func (s *Service) ListClaimLines(ctx context.Context, caseID string, page, perPage int) ([]ClaimLine, int, error) {
	if err := s.cases.RequireActive(ctx, caseID); err != nil {
		return nil, 0, platform.MapNotFound(err, ErrNotFound)
	}
	return s.store.ListClaimLines(ctx, caseID, page, perPage)
}

func (s *Service) CreateClaimLine(ctx context.Context, caseID, rootPersonID, status string, notes platform.NullString) (*ClaimLine, error) {
	var cl *ClaimLine
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.people.RequireActive(ctx, caseID, rootPersonID); err != nil {
			return platform.MapNotFound(err, ErrNotFound)
		}
		var err error
		if cl, err = s.store.CreateClaimLine(ctx, caseID, rootPersonID, status, notes); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "created", cl.ID, nil)
	})
	return cl, err
}

func (s *Service) UpdateClaimLine(ctx context.Context, caseID, lineID string, status *string, notes platform.NullString) (*ClaimLine, error) {
	var cl *ClaimLine
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID, lineID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID, lineID)
		if err != nil {
			return err
		}
		if cl, err = s.store.UpdateClaimLine(ctx, caseID, lineID, status, notes); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "updated", lineID, activitylog.Diff(before.logFields(), cl.logFields()))
	})
	return cl, err
}

func (s *Service) DeleteClaimLine(ctx context.Context, caseID, lineID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID, lineID); err != nil {
			return err
		}
		if err := s.store.DeleteClaimLine(ctx, caseID, lineID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "deleted", lineID, nil)
	})
}

// Restore takes a claim line out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID, lineID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		rootID, err := s.store.TrashedRootPersonID(ctx, caseID, lineID)
		if err != nil {
			return err
		}
		if err := s.people.RequireActive(ctx, caseID, rootID); errors.Is(err, platform.ErrNotFound) {
			return platform.Conflict("This claim line's root ancestor is in the trash. Restore them first.")
		} else if err != nil {
			return err
		}
		if err := s.store.Restore(ctx, caseID, lineID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "restored", lineID, nil)
	})
}

// PermanentDelete deletes a trashed claim line. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID, lineID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		name, err := s.store.LogName(ctx, caseID, lineID)
		if err != nil {
			return err
		}
		if err := s.store.PermanentDelete(ctx, caseID, lineID); err != nil {
			return err
		}
		return s.rec.Named(ctx, caseID, "permanently_deleted", lineID, name, nil)
	})
}

// LogDeletedWithPerson writes a "permanently_deleted" entry for each of the claim lines rooted at a person. The trash
// module calls it inside the permanent delete, before the rows are removed.
func (s *Service) LogDeletedWithPerson(ctx context.Context, caseID, personID string) error {
	ids, err := s.store.IDsRootedAt(ctx, caseID, personID)
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
