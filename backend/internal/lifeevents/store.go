package lifeevents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("life event not found")

type Store interface {
	ListLifeEvents(ctx context.Context, caseID string, page, perPage int) ([]LifeEvent, int, error)
	CreateLifeEvent(ctx context.Context, caseID, personID, eventType string, input UpdateLifeEventInput) (*LifeEvent, error)
	UpdateLifeEvent(ctx context.Context, caseID, eventID string, input UpdateLifeEventInput) (*LifeEvent, error)
	DeleteLifeEvent(ctx context.Context, caseID, eventID string) error
	ReassignLifeEvent(ctx context.Context, caseID, eventID, personID string) (*LifeEvent, error)
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListLifeEvents(ctx context.Context, caseID string, page, perPage int) ([]LifeEvent, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM life_events WHERE case_id = $1 AND deleted_at IS NULL`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count life events: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			le.id::text, le.case_id::text, le.person_id::text,
			le.event_type::text,
			le.event_date::text, le.event_place,
			le.spouse_name, le.spouse_birth_date::text, le.spouse_birth_place,
			le.notes,
			COUNT(d.id) > 0 AS has_documents,
			le.created_at, le.updated_at
		FROM life_events le
		LEFT JOIN documents d ON d.life_event_id = le.id AND d.deleted_at IS NULL
		WHERE le.case_id = $1 AND le.deleted_at IS NULL
		GROUP BY le.id
		ORDER BY le.event_date ASC NULLS LAST, le.created_at ASC
		LIMIT $2 OFFSET $3`,
		caseID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query life events: %w", err)
	}
	defer rows.Close()

	items := []LifeEvent{}
	for rows.Next() {
		le, err := scanLifeEvent(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scan life event: %w", err)
		}
		items = append(items, *le)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate life events: %w", err)
	}

	return items, total, nil
}

func (s *store) CreateLifeEvent(ctx context.Context, caseID, personID, eventType string, input UpdateLifeEventInput) (*LifeEvent, error) {
	if err := s.requirePersonInCase(ctx, caseID, personID); err != nil {
		return nil, err
	}

	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO life_events
			(case_id, person_id, event_type, event_date, event_place,
			 spouse_name, spouse_birth_date, spouse_birth_place, notes)
		VALUES ($1, $2, $3::life_event_type, $4::date, $5, $6, $7::date, $8, $9)
		RETURNING id::text`,
		caseID, personID, eventType,
		nullToSQL(input.EventDate),
		nullToSQL(input.EventPlace),
		nullToSQL(input.SpouseName),
		nullToSQL(input.SpouseBirthDate),
		nullToSQL(input.SpouseBirthPlace),
		nullToSQL(input.Notes),
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert life event: %w", err)
	}

	return s.getLifeEvent(ctx, caseID, id)
}

func (s *store) UpdateLifeEvent(ctx context.Context, caseID, eventID string, input UpdateLifeEventInput) (*LifeEvent, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE life_events SET
			event_type        = COALESCE($3::life_event_type, event_type),
			event_date        = CASE WHEN $4  THEN $5::date  ELSE event_date        END,
			event_place       = CASE WHEN $6  THEN $7        ELSE event_place       END,
			spouse_name       = CASE WHEN $8  THEN $9        ELSE spouse_name       END,
			spouse_birth_date = CASE WHEN $10 THEN $11::date ELSE spouse_birth_date END,
			spouse_birth_place= CASE WHEN $12 THEN $13       ELSE spouse_birth_place END,
			notes             = CASE WHEN $14 THEN $15       ELSE notes             END,
			updated_at        = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		eventID, caseID,
		input.EventType,
		input.EventDate.Set, nullToSQL(input.EventDate),
		input.EventPlace.Set, nullToSQL(input.EventPlace),
		input.SpouseName.Set, nullToSQL(input.SpouseName),
		input.SpouseBirthDate.Set, nullToSQL(input.SpouseBirthDate),
		input.SpouseBirthPlace.Set, nullToSQL(input.SpouseBirthPlace),
		input.Notes.Set, nullToSQL(input.Notes),
	)
	if err != nil {
		return nil, fmt.Errorf("update life event: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}

	return s.getLifeEvent(ctx, caseID, eventID)
}

func (s *store) DeleteLifeEvent(ctx context.Context, caseID, eventID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE life_events SET deleted_at = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		eventID, caseID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete life event: %w", err)
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

func (s *store) ReassignLifeEvent(ctx context.Context, caseID, eventID, personID string) (*LifeEvent, error) {
	if err := s.requirePersonInCase(ctx, caseID, personID); err != nil {
		return nil, err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE life_events SET person_id = $3, updated_at = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		eventID, caseID, personID,
	)
	if err != nil {
		return nil, fmt.Errorf("reassign life event: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}

	return s.getLifeEvent(ctx, caseID, eventID)
}

// getLifeEvent fetches a single non-deleted life event by ID and case, including
// the has_documents flag.
func (s *store) getLifeEvent(ctx context.Context, caseID, eventID string) (*LifeEvent, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			le.id::text, le.case_id::text, le.person_id::text,
			le.event_type::text,
			le.event_date::text, le.event_place,
			le.spouse_name, le.spouse_birth_date::text, le.spouse_birth_place,
			le.notes,
			COUNT(d.id) > 0 AS has_documents,
			le.created_at, le.updated_at
		FROM life_events le
		LEFT JOIN documents d ON d.life_event_id = le.id AND d.deleted_at IS NULL
		WHERE le.id = $1 AND le.case_id = $2 AND le.deleted_at IS NULL
		GROUP BY le.id`,
		eventID, caseID,
	)
	le, err := scanLifeEvent(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get life event: %w", err)
	}
	return le, nil
}

// requirePersonInCase returns ErrNotFound if the person does not exist or does
// not belong to the given case.
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

// scanLifeEvent uses the provided scan function (from either *sql.Row or *sql.Rows)
// to populate a LifeEvent, handling all nullable fields.
func scanLifeEvent(scan func(...any) error) (*LifeEvent, error) {
	var le LifeEvent
	var eventDate, eventPlace, spouseName, spouseBirthDate, spouseBirthPlace, notes sql.NullString
	if err := scan(
		&le.ID, &le.CaseID, &le.PersonID,
		&le.EventType,
		&eventDate, &eventPlace,
		&spouseName, &spouseBirthDate, &spouseBirthPlace,
		&notes,
		&le.HasDocuments,
		&le.CreatedAt, &le.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if eventDate.Valid {
		le.EventDate = &eventDate.String
	}
	if eventPlace.Valid {
		le.EventPlace = &eventPlace.String
	}
	if spouseName.Valid {
		le.SpouseName = &spouseName.String
	}
	if spouseBirthDate.Valid {
		le.SpouseBirthDate = &spouseBirthDate.String
	}
	if spouseBirthPlace.Valid {
		le.SpouseBirthPlace = &spouseBirthPlace.String
	}
	if notes.Valid {
		le.Notes = &notes.String
	}
	return &le, nil
}

func nullToSQL(f NullableField) sql.NullString {
	return sql.NullString{String: f.Value, Valid: f.Valid}
}
