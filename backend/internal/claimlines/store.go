package claimlines

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	ListClaimLines(ctx context.Context, caseID string, page, perPage int) ([]ClaimLine, int, error)
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
		`SELECT COUNT(*) FROM claim_lines WHERE case_id = $1`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count claim lines: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, case_id::text, root_person_id::text, status::text, notes, created_at, updated_at
		FROM claim_lines
		WHERE case_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`,
		caseID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query claim lines: %w", err)
	}
	defer rows.Close()

	items := []ClaimLine{}
	for rows.Next() {
		var cl ClaimLine
		var notes sql.NullString
		if err := rows.Scan(
			&cl.ID, &cl.CaseID, &cl.RootPersonID, &cl.Status, &notes,
			&cl.CreatedAt, &cl.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan claim line: %w", err)
		}
		if notes.Valid {
			cl.Notes = &notes.String
		}
		items = append(items, cl)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate claim lines: %w", err)
	}

	return items, total, nil
}
