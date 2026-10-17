// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package claimlines

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

type Service struct {
	store  Store
	db     *sql.DB
	log    *activitylog.Service
	people PersonChecker
}

func NewService(store Store, db *sql.DB, log *activitylog.Service, people PersonChecker) *Service {
	return &Service{store: store, db: db, log: log, people: people}
}

func (s *Service) ListClaimLines(ctx context.Context, caseID string, page, perPage int) ([]ClaimLine, int, error) {
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
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "created", EntityType: "claim_line",
			EntityID: cl.ID, EntityName: cl.Status + " claim line",
		})
	})
	return cl, err
}

func (s *Service) UpdateClaimLine(ctx context.Context, caseID, lineID string, status *string, notes platform.NullString) (*ClaimLine, error) {
	var cl *ClaimLine
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		var err error
		if cl, err = s.store.UpdateClaimLine(ctx, caseID, lineID, status, notes); err != nil {
			return err
		}
		changes := activitylog.AddString(nil, "status", status)
		changes = activitylog.AddNullable(changes, "notes", notes)
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "updated", EntityType: "claim_line",
			EntityID: cl.ID, EntityName: cl.Status + " claim line", Changes: changes,
		})
	})
	return cl, err
}

func (s *Service) DeleteClaimLine(ctx context.Context, caseID, lineID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.DeleteClaimLine(ctx, caseID, lineID); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "deleted", EntityType: "claim_line",
			EntityID: lineID, EntityName: lineID,
		})
	})
}

// Restore takes a claim line out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID, lineID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.Restore(ctx, caseID, lineID)
	})
}

// PermanentDelete deletes a trashed claim line. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID, lineID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.PermanentDelete(ctx, caseID, lineID)
	})
}
