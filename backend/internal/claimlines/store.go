package claimlines

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("claim line not found")

var validStatuses = map[string]bool{
	"not_yet_researched": true,
	"researching":        true,
	"paused":             true,
	"ineligible":         true,
	"eligible":           true,
}

type Store interface {
	ListClaimLines(ctx context.Context, caseID string, page, perPage int) ([]ClaimLine, int, error)
	CreateClaimLine(ctx context.Context, caseID, rootPersonID, status string, notes NullableField) (*ClaimLine, error)
	UpdateClaimLine(ctx context.Context, caseID, lineID string, status *string, notes NullableField) (*ClaimLine, error)
	DeleteClaimLine(ctx context.Context, caseID, lineID string) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListClaimLines(ctx context.Context, caseID string, page, perPage int) ([]ClaimLine, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM claim_lines WHERE case_id = $1 AND deleted_at IS NULL`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count claim lines: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, case_id::text, root_person_id::text, status::text, notes, created_at, updated_at
		FROM claim_lines
		WHERE case_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
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

func (s *store) CreateClaimLine(ctx context.Context, caseID, rootPersonID, status string, notes NullableField) (*ClaimLine, error) {
	if err := s.requirePersonInCase(ctx, caseID, rootPersonID); err != nil {
		return nil, err
	}

	var id string
	err := s.db.QueryRowContext(ctx, `
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

func (s *store) UpdateClaimLine(ctx context.Context, caseID, lineID string, status *string, notes NullableField) (*ClaimLine, error) {
	result, err := s.db.ExecContext(ctx, `
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
	result, err := s.db.ExecContext(ctx, `
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
	row := s.db.QueryRowContext(ctx, `
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

func (s *store) requirePersonInCase(ctx context.Context, caseID, personID string) error {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM people WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		personID, caseID,
	).Scan(&count); err != nil {
		return fmt.Errorf("verify person: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
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

func nullToSQL(f NullableField) sql.NullString {
	return sql.NullString{String: f.Value, Valid: f.Valid}
}
