// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package lifeevents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/tfior/doc-tracker/platform"
)

var ErrNotFound = fmt.Errorf("life event %w", platform.ErrNotFound)

type Store interface {
	DeletionImpact(ctx context.Context, caseID, eventID string) (*DeletionImpact, error)
	TrashedDeletionImpact(ctx context.Context, caseID, eventID string) (*DeletionImpact, error)
	IDsForPerson(ctx context.Context, caseID, personID string) ([]string, error)
	Get(ctx context.Context, caseID, id string) (*LifeEvent, error)
	LogName(ctx context.Context, caseID, id string) (string, error)
	LockActive(ctx context.Context, caseID, id string) (personID string, err error)
	ListLifeEvents(ctx context.Context, caseID string) ([]LifeEvent, error)
	CreateLifeEvent(ctx context.Context, caseID, personID, eventType string, input UpdateLifeEventInput) (*LifeEvent, error)
	UpdateLifeEvent(ctx context.Context, caseID, eventID string, input UpdateLifeEventInput) (*LifeEvent, error)
	DeleteLifeEvent(ctx context.Context, caseID, eventID string) error
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

func (s *store) ListLifeEvents(ctx context.Context, caseID string) ([]LifeEvent, error) {
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT
			le.id::text, le.case_id::text, le.person_id::text,
			le.event_type::text,
			le.event_date, le.event_date_qualifier::text, le.event_date_end, le.event_place,
			le.spouse_name,
			le.spouse_birth_date, le.spouse_birth_date_qualifier::text, le.spouse_birth_date_end,
			le.spouse_birth_place,
			le.notes,
			COUNT(d.id) > 0 AS has_documents,
			le.version, le.created_at, le.updated_at
		FROM life_events le
		LEFT JOIN active_documents d ON d.life_event_id = le.id
		WHERE le.id IN (SELECT id FROM active_life_events WHERE case_id = $1)
		GROUP BY le.id
		ORDER BY le.event_date COLLATE "C" ASC NULLS LAST, le.created_at ASC, le.id ASC`,
		caseID)
	if err != nil {
		return nil, fmt.Errorf("query life events: %w", err)
	}
	defer rows.Close()

	items := []LifeEvent{}
	for rows.Next() {
		le, err := scanLifeEvent(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan life event: %w", err)
		}
		items = append(items, *le)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate life events: %w", err)
	}

	return items, nil
}

func (s *store) CreateLifeEvent(ctx context.Context, caseID, personID, eventType string, input UpdateLifeEventInput) (*LifeEvent, error) {

	var id string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		INSERT INTO life_events
			(case_id, person_id, event_type,
			 event_date, event_date_qualifier, event_date_end, event_place,
			 spouse_name, spouse_birth_date, spouse_birth_date_qualifier, spouse_birth_date_end,
			 spouse_birth_place, notes)
		VALUES ($1, $2, $3::life_event_type, $4, $5::date_qualifier, $6, $7, $8, $9, $10::date_qualifier, $11, $12, $13)
		RETURNING id::text`,
		caseID, personID, eventType,
		dateText(input.EventDate), dateQualifier(input.EventDate), dateEnd(input.EventDate),
		nullToSQL(input.EventPlace),
		nullToSQL(input.SpouseName),
		dateText(input.SpouseBirthDate), dateQualifier(input.SpouseBirthDate), dateEnd(input.SpouseBirthDate),
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
			person_id         = COALESCE($20::uuid, person_id),
			event_date                  = CASE WHEN $4  THEN $5                   ELSE event_date                  END,
			event_date_qualifier        = CASE WHEN $4  THEN $6::date_qualifier   ELSE event_date_qualifier        END,
			event_date_end              = CASE WHEN $4  THEN $7                   ELSE event_date_end              END,
			event_place                 = CASE WHEN $8  THEN $9                   ELSE event_place                 END,
			spouse_name                 = CASE WHEN $10 THEN $11                  ELSE spouse_name                 END,
			spouse_birth_date           = CASE WHEN $12 THEN $13                  ELSE spouse_birth_date           END,
			spouse_birth_date_qualifier = CASE WHEN $12 THEN $14::date_qualifier  ELSE spouse_birth_date_qualifier END,
			spouse_birth_date_end       = CASE WHEN $12 THEN $15                  ELSE spouse_birth_date_end       END,
			spouse_birth_place          = CASE WHEN $16 THEN $17                  ELSE spouse_birth_place          END,
			notes                       = CASE WHEN $18 THEN $19                  ELSE notes                       END,
			updated_at = NOW(), version = version + 1
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		eventID, caseID,
		input.EventType,
		input.EventDate.Set, dateText(input.EventDate), dateQualifier(input.EventDate), dateEnd(input.EventDate),
		input.EventPlace.Set, nullToSQL(input.EventPlace),
		input.SpouseName.Set, nullToSQL(input.SpouseName),
		input.SpouseBirthDate.Set, dateText(input.SpouseBirthDate), dateQualifier(input.SpouseBirthDate), dateEnd(input.SpouseBirthDate),
		input.SpouseBirthPlace.Set, nullToSQL(input.SpouseBirthPlace),
		input.Notes.Set, nullToSQL(input.Notes),
		input.PersonID,
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
		UPDATE life_events SET deleted_at = NOW(), version = version + 1
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

// getLifeEvent fetches a single non-deleted life event by ID and case, including
// the has_documents flag.
func (s *store) getLifeEvent(ctx context.Context, caseID, eventID string) (*LifeEvent, error) {
	row := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT
			le.id::text, le.case_id::text, le.person_id::text,
			le.event_type::text,
			le.event_date, le.event_date_qualifier::text, le.event_date_end, le.event_place,
			le.spouse_name,
			le.spouse_birth_date, le.spouse_birth_date_qualifier::text, le.spouse_birth_date_end,
			le.spouse_birth_place,
			le.notes,
			COUNT(d.id) > 0 AS has_documents,
			le.version, le.created_at, le.updated_at
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
	var eventPlace, spouseName, spouseBirthPlace, notes sql.NullString
	var eventDate, spouseBirthDate [3]sql.NullString
	if err := scan(
		&le.ID, &le.CaseID, &le.PersonID,
		&le.EventType,
		&eventDate[0], &eventDate[1], &eventDate[2], &eventPlace,
		&spouseName,
		&spouseBirthDate[0], &spouseBirthDate[1], &spouseBirthDate[2],
		&spouseBirthPlace,
		&notes,
		&le.HasDocuments,
		&le.Version, &le.CreatedAt, &le.UpdatedAt,
	); err != nil {
		return nil, err
	}
	le.EventDate = platform.ScanQualifiedDate(eventDate[0], eventDate[1], eventDate[2])
	if eventPlace.Valid {
		le.EventPlace = &eventPlace.String
	}
	if spouseName.Valid {
		le.SpouseName = &spouseName.String
	}
	le.SpouseBirthDate = platform.ScanQualifiedDate(spouseBirthDate[0], spouseBirthDate[1], spouseBirthDate[2])
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

// dateText, dateQualifier and dateEnd are the three columns a qualified date
// is stored in. All three are null when the date is.
func dateText(n platform.NullQualifiedDate) sql.NullString {
	return sql.NullString{String: n.Date.Value, Valid: n.Valid}
}

func dateQualifier(n platform.NullQualifiedDate) sql.NullString {
	return sql.NullString{String: n.Date.Qualifier, Valid: n.Valid}
}

func dateEnd(n platform.NullQualifiedDate) sql.NullString {
	if !n.Valid || n.Date.End == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *n.Date.End, Valid: true}
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
		`UPDATE life_events SET deleted_at = NULL, version = version + 1 WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
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

// Get returns an active record, for the "before" side of an update's log entry.
func (s *store) Get(ctx context.Context, caseID, id string) (*LifeEvent, error) {
	return s.getLifeEvent(ctx, caseID, id)
}

// LogName is the record's name for the activity log. It reads trashed records
// too, so restores and permanent deletes can be named.
func (s *store) LogName(ctx context.Context, caseID, id string) (string, error) {
	var eventType, person string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT le.event_type::text, p.first_name || ' ' || p.last_name
		FROM life_events le JOIN people p ON p.id = le.person_id
		WHERE le.id = $1 AND le.case_id = $2`, id, caseID,
	).Scan(&eventType, &person)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("log name: %w", err)
	}
	return eventTypeLabel(eventType) + " life event for " + person, nil
}

// eventTypeLabel turns an event type into the form used in log names:
// "naturalization" becomes "Naturalization".
func eventTypeLabel(eventType string) string {
	if eventType == "" {
		return eventType
	}
	return strings.ToUpper(eventType[:1]) + strings.ReplaceAll(eventType[1:], "_", " ")
}

// IDsForPerson returns the IDs of a person's life events, trashed ones included.
func (s *store) IDsForPerson(ctx context.Context, caseID, personID string) ([]string, error) {
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx,
		`SELECT id::text FROM life_events WHERE case_id = $1 AND person_id = $2 ORDER BY id`, caseID, personID)
	if err != nil {
		return nil, fmt.Errorf("IDsForPerson: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("IDsForPerson: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeletionImpact counts the active documents under a life event, which would
// go with it.
func (s *store) DeletionImpact(ctx context.Context, caseID, eventID string) (*DeletionImpact, error) {
	var d DeletionImpact
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM active_documents WHERE case_id = $1 AND life_event_id = $2`,
		caseID, eventID,
	).Scan(&d.Documents)
	if err != nil {
		return nil, fmt.Errorf("deletion impact: %w", err)
	}
	return &d, nil
}

// TrashedDeletionImpact counts every document a permanent delete of a life
// event in the trash would remove, including any moved to the trash
// separately. It returns ErrNotFound unless the event is itself in the trash.
func (s *store) TrashedDeletionImpact(ctx context.Context, caseID, eventID string) (*DeletionImpact, error) {
	var d DeletionImpact
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM documents WHERE life_event_id = le.id)
		FROM life_events le
		WHERE le.id = $2 AND le.case_id = $1 AND le.deleted_at IS NOT NULL`,
		caseID, eventID,
	).Scan(&d.Documents)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("deletion impact: %w", err)
	}
	return &d, nil
}
