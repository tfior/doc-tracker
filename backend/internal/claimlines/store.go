// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package claimlines

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/tfior/doc-tracker/platform"
)

var ErrNotFound = fmt.Errorf("claim line %w", platform.ErrNotFound)

var validStatuses = map[string]bool{
	"not_yet_researched": true,
	"researching":        true,
	"paused":             true,
	"ineligible":         true,
	"eligible":           true,
}

type Store interface {
	IDsRootedAt(ctx context.Context, caseID, personID string) ([]string, error)
	Get(ctx context.Context, caseID, id string) (*ClaimLine, error)
	LogName(ctx context.Context, caseID, id string) (string, error)
	LockActive(ctx context.Context, caseID, id string) error
	ListClaimLines(ctx context.Context, caseID string, page, perPage int) ([]ClaimLine, int, error)
	CreateClaimLine(ctx context.Context, caseID, rootPersonID, status string, notes platform.NullString) (*ClaimLine, error)
	UpdateClaimLine(ctx context.Context, caseID, lineID string, status *string, notes platform.NullString) (*ClaimLine, error)
	DeleteClaimLine(ctx context.Context, caseID, lineID string) error
	RequireActive(ctx context.Context, caseID, lineID string) error
	TrashedRootPersonID(ctx context.Context, caseID, lineID string) (string, error)
	Restore(ctx context.Context, caseID, id string) error
	PermanentDelete(ctx context.Context, caseID, id string) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListClaimLines(ctx context.Context, caseID string, page, perPage int) ([]ClaimLine, int, error) {
	var total int
	if err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM active_claim_lines WHERE case_id = $1`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count claim lines: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT id::text, case_id::text, root_person_id::text, status::text, notes, created_at, updated_at
		FROM claim_lines
		WHERE id IN (SELECT id FROM active_claim_lines WHERE case_id = $1)
		ORDER BY created_at ASC, id ASC
		LIMIT $2 OFFSET $3`,
		caseID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query claim lines: %w", err)
	}
	defer rows.Close()

	items := []ClaimLine{}
	for rows.Next() {
		cl, err := scanClaimLine(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scan claim line: %w", err)
		}
		items = append(items, *cl)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate claim lines: %w", err)
	}

	return items, total, nil
}

func (s *store) CreateClaimLine(ctx context.Context, caseID, rootPersonID, status string, notes platform.NullString) (*ClaimLine, error) {

	var id string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		INSERT INTO claim_lines (case_id, root_person_id, status, notes)
		VALUES ($1, $2, $3::claim_line_status, $4)
		RETURNING id::text`,
		caseID, rootPersonID, status, nullToSQL(notes),
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert claim line: %w", err)
	}

	return s.getClaimLine(ctx, caseID, id)
}

func (s *store) UpdateClaimLine(ctx context.Context, caseID, lineID string, status *string, notes platform.NullString) (*ClaimLine, error) {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
		UPDATE claim_lines SET
			status     = COALESCE($3::claim_line_status, status),
			notes      = CASE WHEN $4 THEN $5 ELSE notes END,
			updated_at = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		lineID, caseID,
		status,
		notes.Set, nullToSQL(notes),
	)
	if err != nil {
		return nil, fmt.Errorf("update claim line: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}

	return s.getClaimLine(ctx, caseID, lineID)
}

func (s *store) DeleteClaimLine(ctx context.Context, caseID, lineID string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
		UPDATE claim_lines SET deleted_at = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		lineID, caseID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete claim line: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *store) getClaimLine(ctx context.Context, caseID, lineID string) (*ClaimLine, error) {
	row := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT id::text, case_id::text, root_person_id::text, status::text, notes, created_at, updated_at
		FROM claim_lines
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		lineID, caseID,
	)
	cl, err := scanClaimLine(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get claim line: %w", err)
	}
	return cl, nil
}

func scanClaimLine(scan func(...any) error) (*ClaimLine, error) {
	var cl ClaimLine
	var notes sql.NullString
	if err := scan(
		&cl.ID, &cl.CaseID, &cl.RootPersonID, &cl.Status, &notes,
		&cl.CreatedAt, &cl.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if notes.Valid {
		cl.Notes = &notes.String
	}
	return &cl, nil
}

func nullToSQL(f platform.NullString) sql.NullString {
	return sql.NullString{String: f.Value, Valid: f.Valid}
}

func (s *store) Restore(ctx context.Context, caseID, id string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`UPDATE claim_lines SET deleted_at = NULL WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		id, caseID)
	if err != nil {
		return fmt.Errorf("restore claim line: %w", err)
	}
	return platform.RequireOneRow(result, ErrNotFound)
}

func (s *store) PermanentDelete(ctx context.Context, caseID, id string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`DELETE FROM claim_lines WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		id, caseID)
	if err != nil {
		return fmt.Errorf("permanent delete claim line: %w", err)
	}
	return platform.RequireOneRow(result, ErrNotFound)
}

// RequireActive returns ErrNotFound unless the claim line is active: not in
// the trash and its root ancestor active. It locks the line, its root ancestor
// and its case until the transaction ends.
func (s *store) RequireActive(ctx context.Context, caseID, lineID string) error {
	var id string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT id::text FROM active_claim_lines WHERE id = $1 AND case_id = $2 FOR SHARE`,
		lineID, caseID,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("verify claim line: %w", err)
	}
	return nil
}

// TrashedRootPersonID returns the root ancestor of a claim line that is in the
// trash, or ErrNotFound if the line is not in the trash.
func (s *store) TrashedRootPersonID(ctx context.Context, caseID, lineID string) (string, error) {
	var personID string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT root_person_id::text FROM claim_lines WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		lineID, caseID,
	).Scan(&personID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get trashed claim line: %w", err)
	}
	return personID, nil
}

// LockActive takes an exclusive lock on a row about to be changed, then checks
// that it is active. Taking the exclusive lock first means simultaneous writes
// to the same row wait for each other instead of deadlocking.
func (s *store) LockActive(ctx context.Context, caseID, id string) error {
	if _, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`SELECT id FROM claim_lines WHERE id = $1 FOR NO KEY UPDATE`, id); err != nil {
		return fmt.Errorf("lock claim_lines: %w", err)
	}
	return s.RequireActive(ctx, caseID, id)
}

// Get returns an active record, for the "before" side of an update's log entry.
func (s *store) Get(ctx context.Context, caseID, id string) (*ClaimLine, error) {
	return s.getClaimLine(ctx, caseID, id)
}

// LogName is the record's name for the activity log. It reads trashed records
// too, so restores and permanent deletes can be named.
func (s *store) LogName(ctx context.Context, caseID, id string) (string, error) {
	var root string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT p.first_name || ' ' || p.last_name
		FROM claim_lines cl JOIN people p ON p.id = cl.root_person_id
		WHERE cl.id = $1 AND cl.case_id = $2`, id, caseID,
	).Scan(&root)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("log name: %w", err)
	}
	return "Claim line through " + root, nil
}

// IDsRootedAt returns the IDs of the claim lines rooted at a person, trashed ones included.
func (s *store) IDsRootedAt(ctx context.Context, caseID, personID string) ([]string, error) {
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx,
		`SELECT id::text FROM claim_lines WHERE case_id = $1 AND root_person_id = $2 ORDER BY id`, caseID, personID)
	if err != nil {
		return nil, fmt.Errorf("IDsRootedAt: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("IDsRootedAt: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
