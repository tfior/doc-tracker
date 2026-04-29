package cases

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("case not found")

type Store interface {
	ListCases(ctx context.Context, page, perPage int) ([]Case, int, error)
	GetCase(ctx context.Context, caseID string) (*CaseDetail, error)
	CreateCase(ctx context.Context, title string) (*Case, error)
	UpdateCase(ctx context.Context, caseID string, title *string, status *string) (*Case, error)
	DeleteCase(ctx context.Context, caseID string) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListCases(ctx context.Context, page, perPage int) ([]Case, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cases WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count cases: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, title, status::text, primary_root_person_id::text, created_at, updated_at
		FROM cases
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query cases: %w", err)
	}
	defer rows.Close()

	items := []Case{}
	for rows.Next() {
		var c Case
		var primaryRootPersonID sql.NullString
		if err := rows.Scan(&c.ID, &c.Title, &c.Status, &primaryRootPersonID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan case: %w", err)
		}
		if primaryRootPersonID.Valid {
			c.PrimaryRootPersonID = &primaryRootPersonID.String
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate cases: %w", err)
	}

	return items, total, nil
}

func (s *store) GetCase(ctx context.Context, caseID string) (*CaseDetail, error) {
	var d CaseDetail
	var primaryRootPersonID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT
			c.id::text,
			c.title,
			c.status::text,
			c.primary_root_person_id::text,
			c.created_at,
			c.updated_at,
			COUNT(DISTINCT cl.id) FILTER (WHERE cl.status = 'not_yet_researched') AS cl_not_yet_researched,
			COUNT(DISTINCT cl.id) FILTER (WHERE cl.status = 'researching')       AS cl_researching,
			COUNT(DISTINCT cl.id) FILTER (WHERE cl.status = 'paused')            AS cl_paused,
			COUNT(DISTINCT cl.id) FILTER (WHERE cl.status = 'ineligible')        AS cl_ineligible,
			COUNT(DISTINCT cl.id) FILTER (WHERE cl.status = 'eligible')          AS cl_eligible,
			COUNT(DISTINCT d.id)  FILTER (WHERE ds.progress_bucket = 'not_started') AS doc_not_started,
			COUNT(DISTINCT d.id)  FILTER (WHERE ds.progress_bucket = 'in_progress') AS doc_in_progress,
			COUNT(DISTINCT d.id)  FILTER (WHERE ds.progress_bucket = 'complete')    AS doc_complete
		FROM cases c
		LEFT JOIN claim_lines cl ON cl.case_id = c.id
		LEFT JOIN documents d ON d.case_id = c.id
		LEFT JOIN document_statuses ds ON ds.id = d.status_id
		WHERE c.id = $1 AND c.deleted_at IS NULL
		GROUP BY c.id`,
		caseID,
	).Scan(
		&d.ID, &d.Title, &d.Status, &primaryRootPersonID, &d.CreatedAt, &d.UpdatedAt,
		&d.ClaimLineSummary.NotYetResearched, &d.ClaimLineSummary.Researching,
		&d.ClaimLineSummary.Paused, &d.ClaimLineSummary.Ineligible, &d.ClaimLineSummary.Eligible,
		&d.DocumentProgress.NotStarted, &d.DocumentProgress.InProgress, &d.DocumentProgress.Complete,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query case: %w", err)
	}

	if primaryRootPersonID.Valid {
		d.PrimaryRootPersonID = &primaryRootPersonID.String
	}
	d.ClaimLineSummary.Total = d.ClaimLineSummary.NotYetResearched + d.ClaimLineSummary.Researching +
		d.ClaimLineSummary.Paused + d.ClaimLineSummary.Ineligible + d.ClaimLineSummary.Eligible

	return &d, nil
}

func (s *store) CreateCase(ctx context.Context, title string) (*Case, error) {
	var c Case
	var primaryRootPersonID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO cases (title)
		VALUES ($1)
		RETURNING id::text, title, status::text, primary_root_person_id::text, created_at, updated_at`,
		title,
	).Scan(&c.ID, &c.Title, &c.Status, &primaryRootPersonID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert case: %w", err)
	}
	if primaryRootPersonID.Valid {
		c.PrimaryRootPersonID = &primaryRootPersonID.String
	}
	return &c, nil
}

func (s *store) UpdateCase(ctx context.Context, caseID string, title *string, status *string) (*Case, error) {
	var c Case
	var primaryRootPersonID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		UPDATE cases
		SET
			title      = COALESCE($2, title),
			status     = COALESCE($3::case_status, status),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, title, status::text, primary_root_person_id::text, created_at, updated_at`,
		caseID, title, status,
	).Scan(&c.ID, &c.Title, &c.Status, &primaryRootPersonID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update case: %w", err)
	}
	if primaryRootPersonID.Valid {
		c.PrimaryRootPersonID = &primaryRootPersonID.String
	}
	return &c, nil
}

func (s *store) DeleteCase(ctx context.Context, caseID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE cases SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		caseID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete case: %w", err)
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
