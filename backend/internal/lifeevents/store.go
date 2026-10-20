// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package lifeevents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/tfior/doc-tracker/platform"
)

var ErrNotFound = fmt.Errorf("life event %w", platform.ErrNotFound)

type Store interface {
	LockActive(ctx context.Context, caseID, id string) (personID string, err error)
	ListLifeEvents(ctx context.Context, caseID string, page, perPage int) ([]LifeEvent, int, error)
	CreateLifeEvent(ctx context.Context, caseID, personID, eventType string, input UpdateLifeEventInput) (*LifeEvent, error)
	UpdateLifeEvent(ctx context.Context, caseID, eventID string, input UpdateLifeEventInput) (*LifeEvent, error)
	DeleteLifeEvent(ctx context.Context, caseID, eventID string) error
	ReassignLifeEvent(ctx context.Context, caseID, eventID, personID string) (*LifeEvent, error)
	RequireActive(ctx context.Context, caseID, eventID string) (personID string, err error)
	TrashedPersonID(ctx context.Context, caseID, eventID string) (string, error)
	Restore(ctx context.Context, caseID, id string) error
	PermanentDelete(ctx context.Context, caseID, id string) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListLifeEvents(ctx context.Context, caseID string, page, perPage int) ([]LifeEvent, int, error) {
	var total int
	if err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM active_life_events WHERE case_id = $1`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count life events: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT
			le.id::text, le.case_id::text, le.person_id::text,
			le.event_type::text,
			le.event_date::text, le.event_place,
			le.spouse_name, le.spouse_birth_date::text, le.spouse_birth_place,
			le.notes,
			COUNT(d.id) > 0 AS has_documents,
			le.created_at, le.updated_at
		FROM life_events le
		LEFT JOIN active_documents d ON d.life_event_id = le.id
		WHERE le.id IN (SELECT id FROM active_life_events WHERE case_id = $1)
		GROUP BY le.id
		ORDER BY le.event_date ASC NULLS LAST, le.created_at ASC, le.id ASC
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

	var id string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
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
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
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
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
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

	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
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
	row := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT
			le.id::text, le.case_id::text, le.person_id::text,
			le.event_type::text,
			le.event_date::text, le.event_place,
			le.spouse_name, le.spouse_birth_date::text, le.spouse_birth_place,
			le.notes,
			COUNT(d.id) > 0 AS has_documents,
			le.created_at, le.updated_at
		FROM life_events le
		LEFT JOIN active_documents d ON d.life_event_id = le.id
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

func nullToSQL(f platform.NullString) sql.NullString {
	return sql.NullString{String: f.Value, Valid: f.Valid}
}

// RequireActive returns the event's person, or ErrNotFound unless the event is
// active: not in the trash and not beneath a trashed person or case. It locks
// the event, its person and its case until the transaction ends.
func (s *store) RequireActive(ctx context.Context, caseID, eventID string) (string, error) {
	var personID string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT person_id::text FROM active_life_events WHERE id = $1 AND case_id = $2 FOR SHARE`,
		eventID, caseID,
	).Scan(&personID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("verify life event: %w", err)
	}
	return personID, nil
}

// TrashedPersonID returns the person of a life event that is in the trash, or
// ErrNotFound if the event is not in the trash.
func (s *store) TrashedPersonID(ctx context.Context, caseID, eventID string) (string, error) {
	var personID string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT person_id::text FROM life_events WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		eventID, caseID,
	).Scan(&personID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get trashed life event: %w", err)
	}
	return personID, nil
}

func (s *store) Restore(ctx context.Context, caseID, id string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`UPDATE life_events SET deleted_at = NULL WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		id, caseID)
	if err != nil {
		return fmt.Errorf("restore life event: %w", err)
	}
	return platform.RequireOneRow(result, ErrNotFound)
}

func (s *store) PermanentDelete(ctx context.Context, caseID, id string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`DELETE FROM life_events WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		id, caseID)
	if err != nil {
		return fmt.Errorf("permanent delete life event: %w", err)
	}
	return platform.RequireOneRow(result, ErrNotFound)
}

// LockActive takes an exclusive lock on a row about to be changed, then checks
// that it is active. Taking the exclusive lock first means simultaneous writes
// to the same row wait for each other instead of deadlocking.
func (s *store) LockActive(ctx context.Context, caseID, id string) (string, error) {
	if _, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`SELECT id FROM life_events WHERE id = $1 FOR NO KEY UPDATE`, id); err != nil {
		return "", fmt.Errorf("lock life_events: %w", err)
	}
	return s.RequireActive(ctx, caseID, id)
}
