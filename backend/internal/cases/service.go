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
	store    Store
	progress ProgressCounter
	db       *sql.DB
	log      *activitylog.Service
	rec      *activitylog.Recorder
}

func NewService(store Store, db *sql.DB, log *activitylog.Service) *Service {
	return &Service{store: store, db: db, log: log, rec: log.For("case",
		func(ctx context.Context, _, caseID string) (string, error) { return store.LogName(ctx, caseID) })}
}

// SetProgressCounter supplies the documents module's progress counts. It is
// set after construction because the documents service depends on this one.
func (s *Service) SetProgressCounter(p ProgressCounter) {
	s.progress = p
}

func (s *Service) ListCases(ctx context.Context, page, perPage int) ([]CaseSummary, int, error) {
	list, total, err := s.store.ListCases(ctx, page, perPage)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, len(list))
	for i, c := range list {
		ids[i] = c.ID
	}
	progress, err := s.progress.Progress(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	items := make([]CaseSummary, len(list))
	for i, c := range list {
		items[i] = CaseSummary{Case: c, Progress: progress[c.ID]}
	}
	return items, total, nil
}

func (s *Service) GetCase(ctx context.Context, caseID string) (*CaseDetail, error) {
	d, err := s.store.GetCase(ctx, caseID)
	if err != nil {
		return nil, err
	}
	progress, err := s.progress.Progress(ctx, []string{caseID})
	if err != nil {
		return nil, err
	}
	d.Progress = progress[caseID]
	return d, nil
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
		return s.rec.Record(ctx, c.ID, "created", c.ID, nil)
	})
	return c, err
}

func (s *Service) UpdateCase(ctx context.Context, caseID string, version int, title *string, status *string) (*Case, error) {
	var c *Case
	err := platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID)
		if err != nil {
			return err
		}
		if err := platform.CheckVersion(before.Version, version); err != nil {
			return err
		}
		if c, err = s.store.UpdateCase(ctx, caseID, title, status); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "updated", caseID, activitylog.Diff(before.logFields(), c.logFields()))
	})
	return c, err
}

func (s *Service) DeleteCase(ctx context.Context, caseID string, version int) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.LockActive(ctx, caseID); err != nil {
			return err
		}
		before, err := s.store.Get(ctx, caseID)
		if err != nil {
			return err
		}
		if err := platform.CheckVersion(before.Version, version); err != nil {
			return err
		}
		if err := s.store.DeleteCase(ctx, caseID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "deleted", caseID, nil)
	})
}

// Restore takes a case out of the trash. Called by the trash module.
func (s *Service) Restore(ctx context.Context, caseID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		if err := s.store.RestoreCase(ctx, caseID); err != nil {
			return err
		}
		return s.rec.Record(ctx, caseID, "restored", caseID, nil)
	})
}

// PermanentDelete deletes a trashed case and everything in it. Called by the trash module.
func (s *Service) PermanentDelete(ctx context.Context, caseID string) error {
	return platform.WithTx(ctx, s.db, func(ctx context.Context) error {
		return s.store.PermanentDeleteCase(ctx, caseID)
	})
}
