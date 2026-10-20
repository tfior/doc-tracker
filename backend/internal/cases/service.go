// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cases

import (
	"context"
	"database/sql"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/platform"
)

type Service struct {
	store Store
	db    *sql.DB
	log   *activitylog.Service
}

func NewService(store Store, db *sql.DB, log *activitylog.Service) *Service {
	return &Service{store: store, db: db, log: log}
}

func (s *Service) ListCases(ctx context.Context, page, perPage int) ([]Case, int, error) {
	return s.store.ListCases(ctx, page, perPage)
}

func (s *Service) GetCase(ctx context.Context, caseID string) (*CaseDetail, error) {
	return s.store.GetCase(ctx, caseID)
}

// RequireActive returns an error wrapping platform.ErrNotFound unless the case
// exists and is not in the trash. Other modules call this through an interface.
func (s *Service) RequireActive(ctx context.Context, caseID string) error {
	return s.store.RequireActive(ctx, caseID)
}

func (s *Service) CreateCase(ctx context.Context, title string) (*Case, error) {
	var c *Case
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		var err error
		if c, err = s.store.CreateCase(ctx, title); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: c.ID, Action: "created", EntityType: "case",
			EntityID: c.ID, EntityName: c.Title,
		})
	})
	return c, err
}

func (s *Service) UpdateCase(ctx context.Context, caseID string, title *string, status *string) (*Case, error) {
	var c *Case
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID); err != nil {
			return err
		}
		var err error
		if c, err = s.store.UpdateCase(ctx, caseID, title, status); err != nil {
			return err
		}
		changes := activitylog.AddString(nil, "title", title)
		changes = activitylog.AddString(changes, "status", status)
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: c.ID, Action: "updated", EntityType: "case",
			EntityID: c.ID, EntityName: c.Title, Changes: changes,
		})
	})
	return c, err
}

func (s *Service) DeleteCase(ctx context.Context, caseID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID); err != nil {
			return err
		}
		if err := s.store.DeleteCase(ctx, caseID); err != nil {
			return err
		}
		return s.log.Insert(ctx, activitylog.InsertParams{
			CaseID: caseID, Action: "deleted", EntityType: "case",
			EntityID: caseID, EntityName: caseID,
		})
	})
}

// Restore takes a case out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.RestoreCase(ctx, caseID)
	})
}

// PermanentDelete deletes a trashed case and everything in it. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.PermanentDeleteCase(ctx, caseID)
	})
}
