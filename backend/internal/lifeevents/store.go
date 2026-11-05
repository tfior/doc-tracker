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
	Peek(ctx context.Context, caseID, eventID string) (Peeked, error)
	SharedMarriages(ctx context.Context, caseID, personID string, lock bool) ([]SharedMarriage, error)
	SpouseDetails(ctx context.Context, caseID, personID string) (*SpouseDetails, error)
	HandOver(ctx context.Context, caseID, eventID, personID string, d *SpouseDetails) (before, after *LifeEvent, err error)
	VitalEvent(ctx context.Context, caseID, personID, eventType string) (string, error)
	VitalHolder(ctx context.Context, caseID, personID, eventType, exceptID string) (string, error)
	Restore(ctx context.Context, caseID, id string) error
	PermanentDelete(ctx context.Context, caseID, id string) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

// eventColumns are the columns scanLifeEvent reads, from life_events le joined
// to its active documents d.
const eventColumns = `
			le.id::text, le.case_id::text, le.person_id::text,
			le.event_type::text,
			le.event_date, le.event_date_qualifier::text, le.event_date_end, le.event_place,
			le.spouse_person_id::text, le.spouse_name,
			le.spouse_birth_date, le.spouse_birth_date_qualifier::text, le.spouse_birth_date_end,
			le.spouse_birth_place,
			le.divorce_date, le.divorce_date_qualifier::text, le.divorce_date_end, le.divorce_place,
			le.notes,
			COUNT(d.id) > 0 AS has_documents,
			le.version, le.created_at, le.updated_at`

func (s *store) ListLifeEvents(ctx context.Context, caseID string) ([]LifeEvent, error) {
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT `+eventColumns+`
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
			 spouse_birth_place, notes,
			 spouse_person_id, divorce_date, divorce_date_qualifier, divorce_date_end, divorce_place)
		VALUES ($1, $2, $3::life_event_type, $4, $5::date_qualifier, $6, $7, $8, $9, $10::date_qualifier, $11, $12, $13,
			$14::uuid, $15, $16::date_qualifier, $17, $18)
		RETURNING id::text`,
		caseID, personID, eventType,
		dateText(input.EventDate), dateQualifier(input.EventDate), dateEnd(input.EventDate),
		nullToSQL(input.EventPlace),
		nullToSQL(input.SpouseName),
		dateText(input.SpouseBirthDate), dateQualifier(input.SpouseBirthDate), dateEnd(input.SpouseBirthDate),
		nullToSQL(input.SpouseBirthPlace),
		nullToSQL(input.Notes),
		nullToSQL(input.SpousePersonID),
		dateText(input.DivorceDate), dateQualifier(input.DivorceDate), dateEnd(input.DivorceDate),
		nullToSQL(input.DivorcePlace),
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
			spouse_person_id            = CASE WHEN $21 THEN $22::uuid            ELSE spouse_person_id            END,
			divorce_date                = CASE WHEN $23 THEN $24                  ELSE divorce_date                END,
			divorce_date_qualifier      = CASE WHEN $23 THEN $25::date_qualifier  ELSE divorce_date_qualifier      END,
			divorce_date_end            = CASE WHEN $23 THEN $26                  ELSE divorce_date_end            END,
			divorce_place               = CASE WHEN $27 THEN $28                  ELSE divorce_place               END,
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
		input.SpousePersonID.Set, nullToSQL(input.SpousePersonID),
		input.DivorceDate.Set, dateText(input.DivorceDate), dateQualifier(input.DivorceDate), dateEnd(input.DivorceDate),
		input.DivorcePlace.Set, nullToSQL(input.DivorcePlace),
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
	return s.get(ctx, caseID, eventID, false)
}

// get fetches a life event, and one in the trash too when trashed is true.
func (s *store) get(ctx context.Context, caseID, eventID string, trashed bool) (*LifeEvent, error) {
	row := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT `+eventColumns+`
		FROM life_events le
		LEFT JOIN active_documents d ON d.life_event_id = le.id
		WHERE le.id = $1 AND le.case_id = $2 AND ($3 OR le.deleted_at IS NULL)
		GROUP BY le.id`,
		eventID, caseID, trashed,
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
	var eventPlace, spousePersonID, spouseName, spouseBirthPlace, divorcePlace, notes sql.NullString
	var eventDate, spouseBirthDate, divorceDate [3]sql.NullString
	if err := scan(
		&le.ID, &le.CaseID, &le.PersonID,
		&le.EventType,
		&eventDate[0], &eventDate[1], &eventDate[2], &eventPlace,
		&spousePersonID, &spouseName,
		&spouseBirthDate[0], &spouseBirthDate[1], &spouseBirthDate[2],
		&spouseBirthPlace,
		&divorceDate[0], &divorceDate[1], &divorceDate[2], &divorcePlace,
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
	if spousePersonID.Valid {
		le.SpousePersonID = &spousePersonID.String
	}
	if spouseName.Valid {
		le.SpouseName = &spouseName.String
	}
	le.SpouseBirthDate = platform.ScanQualifiedDate(spouseBirthDate[0], spouseBirthDate[1], spouseBirthDate[2])
	if spouseBirthPlace.Valid {
		le.SpouseBirthPlace = &spouseBirthPlace.String
	}
	le.DivorceDate = platform.ScanQualifiedDate(divorceDate[0], divorceDate[1], divorceDate[2])
	if divorcePlace.Valid {
		le.DivorcePlace = &divorcePlace.String
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

// lockPeopleOf locks the people a life event depends on, its person and linked
// spouse, so neither can be trashed or deleted until the transaction ends.
// People are locked before their events, the order every write takes them.
func (s *store) lockPeopleOf(ctx context.Context, eventID string) error {
	if _, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
		SELECT p.id FROM people p
		JOIN life_events le ON p.id IN (le.person_id, le.spouse_person_id)
		WHERE le.id = $1
		ORDER BY p.id FOR SHARE OF p`, eventID); err != nil {
		return fmt.Errorf("lock life event's people: %w", err)
	}
	return nil
}

// RequireActive returns the event's person, or ErrNotFound unless the event is
// active: not in the trash, in an active case, and with at least one spouse
// active. It locks the event, its people and its case until the transaction
// ends, the people first.
func (s *store) RequireActive(ctx context.Context, caseID, eventID string) (string, error) {
	conn := platform.Conn(ctx, s.db)
	if err := s.lockPeopleOf(ctx, eventID); err != nil {
		return "", err
	}
	var personID string
	err := conn.QueryRowContext(ctx,
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

// Peeked is what Peek reads of a life event.
type Peeked struct {
	PersonID, SpouseID, EventType string // SpouseID is "" without a linked spouse
	Trashed                       bool
}

// Peek reads a life event's person, linked spouse and type, and whether it is
// in the trash, without locking it. It tells a write which people to lock
// before the event itself.
func (s *store) Peek(ctx context.Context, caseID, eventID string) (Peeked, error) {
	var p Peeked
	var spouse sql.NullString
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT person_id::text, spouse_person_id::text, event_type::text, deleted_at IS NOT NULL
		FROM life_events WHERE id = $1 AND case_id = $2`,
		eventID, caseID,
	).Scan(&p.PersonID, &spouse, &p.EventType, &p.Trashed)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, fmt.Errorf("peek life event: %w", err)
	}
	p.SpouseID = spouse.String
	return p, nil
}

// SharedMarriage is a marriage, in the trash or not, with a linked spouse.
type SharedMarriage struct {
	ID, PersonID, SpouseID string
}

// SharedMarriages returns the marriages with a linked spouse that a person is
// part of, as either spouse, including those in the trash. It locks them only
// when lock is true.
func (s *store) SharedMarriages(ctx context.Context, caseID, personID string, lock bool) ([]SharedMarriage, error) {
	query := `
		SELECT id::text, person_id::text, spouse_person_id::text FROM life_events
		WHERE case_id = $1 AND spouse_person_id IS NOT NULL AND $2 IN (person_id, spouse_person_id)
		ORDER BY id`
	if lock {
		query += ` FOR NO KEY UPDATE`
	}
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, query, caseID, personID)
	if err != nil {
		return nil, fmt.Errorf("shared marriages: %w", err)
	}
	defer rows.Close()
	var out []SharedMarriage
	for rows.Next() {
		var m SharedMarriage
		if err := rows.Scan(&m.ID, &m.PersonID, &m.SpouseID); err != nil {
			return nil, fmt.Errorf("scan shared marriage: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SpouseDetails are a person's name, birth date and birth place, as a marriage
// keeps them in its flat spouse fields.
type SpouseDetails struct {
	Name       string
	BirthDate  platform.NullQualifiedDate
	BirthPlace platform.NullString
}

// SpouseDetails reads a person's details for the flat spouse fields: their
// name, and the date and place of their birth event outside the trash.
func (s *store) SpouseDetails(ctx context.Context, caseID, personID string) (*SpouseDetails, error) {
	var name string
	var date [3]sql.NullString
	var place sql.NullString
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT p.first_name || ' ' || p.last_name,
			b.event_date, b.event_date_qualifier::text, b.event_date_end, b.event_place
		FROM people p
		LEFT JOIN life_events b ON b.person_id = p.id AND b.event_type = 'birth' AND b.deleted_at IS NULL
		WHERE p.id = $1 AND p.case_id = $2`,
		personID, caseID,
	).Scan(&name, &date[0], &date[1], &date[2], &place)
	if err != nil {
		return nil, fmt.Errorf("spouse details: %w", err)
	}
	d := &SpouseDetails{Name: name, BirthPlace: platform.NullString{Set: true, Valid: place.Valid, Value: place.String}}
	d.BirthDate.Set = true
	if q := platform.ScanQualifiedDate(date[0], date[1], date[2]); q != nil {
		d.BirthDate.Valid, d.BirthDate.Date = true, *q
	}
	return d, nil
}

// HandOver gives a shared marriage, in the trash or not, to personID with no
// linked spouse, keeping the departing spouse's details in the flat fields.
func (s *store) HandOver(ctx context.Context, caseID, eventID, personID string, d *SpouseDetails) (*LifeEvent, *LifeEvent, error) {
	before, err := s.get(ctx, caseID, eventID, true)
	if err != nil {
		return nil, nil, err
	}
	_, err = platform.Conn(ctx, s.db).ExecContext(ctx, `
		UPDATE life_events SET
			person_id = $3, spouse_person_id = NULL,
			spouse_name = $4,
			spouse_birth_date = $5, spouse_birth_date_qualifier = $6::date_qualifier, spouse_birth_date_end = $7,
			spouse_birth_place = $8,
			updated_at = NOW(), version = version + 1
		WHERE id = $1 AND case_id = $2`,
		eventID, caseID, personID, d.Name,
		dateText(d.BirthDate), dateQualifier(d.BirthDate), dateEnd(d.BirthDate),
		nullToSQL(d.BirthPlace),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("hand over marriage: %w", err)
	}
	after, err := s.get(ctx, caseID, eventID, true)
	return before, after, err
}

// VitalEvent locks and returns the ID of the person's birth or death event
// that is not in the trash, or "" if they have none.
func (s *store) VitalEvent(ctx context.Context, caseID, personID, eventType string) (string, error) {
	var id string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT id::text FROM life_events
		WHERE case_id = $1 AND person_id = $2 AND event_type = $3::life_event_type AND deleted_at IS NULL
		FOR NO KEY UPDATE`,
		caseID, personID, eventType,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("vital event: %w", err)
	}
	return id, nil
}

// VitalHolder returns the person's name if they already have a birth or death
// event, other than exceptID, that is not in the trash; otherwise "".
func (s *store) VitalHolder(ctx context.Context, caseID, personID, eventType, exceptID string) (string, error) {
	var name string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT p.first_name || ' ' || p.last_name
		FROM life_events le JOIN people p ON p.id = le.person_id
		WHERE le.case_id = $1 AND le.person_id = $2 AND le.event_type = $3::life_event_type
		  AND le.deleted_at IS NULL AND le.id::text <> $4
		LIMIT 1`,
		caseID, personID, eventType, exceptID,
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("vital holder: %w", err)
	}
	return name, nil
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
	if err := s.lockPeopleOf(ctx, id); err != nil {
		return "", err
	}
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
	var spouse sql.NullString
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT le.event_type::text, p.first_name || ' ' || p.last_name, sp.first_name || ' ' || sp.last_name
		FROM life_events le
		JOIN people p ON p.id = le.person_id
		LEFT JOIN people sp ON sp.id = le.spouse_person_id
		WHERE le.id = $1 AND le.case_id = $2`, id, caseID,
	).Scan(&eventType, &person, &spouse)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("log name: %w", err)
	}
	if spouse.Valid {
		person += " and " + spouse.String
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
