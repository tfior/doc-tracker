// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package db_test checks that migrations carry existing data forward. Each
// test migrates a scratch database part of the way, adds data in the old
// shape, then migrates to the end and reads it back.
package db_test

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scratchDB = "doctracker_migrations_test"

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func dsn(dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("DB_USER", "doctracker"), getenv("DB_PASSWORD", "changeme"),
		getenv("DB_HOST", "localhost"), getenv("DB_PORT", "5432"), dbName)
}

// migrateTo recreates the scratch database, migrates it to version, and
// returns a connection to it and a migrator for the remaining steps.
func migrateTo(t *testing.T, version uint) (*sql.DB, *migrate.Migrate) {
	t.Helper()
	admin, err := sql.Open("postgres", dsn("postgres"))
	require.NoError(t, err)
	defer admin.Close()
	_, err = admin.Exec(`DROP DATABASE IF EXISTS ` + scratchDB + ` WITH (FORCE)`)
	require.NoError(t, err, "cannot reach the database — run 'make dev' first")
	_, err = admin.Exec(`CREATE DATABASE ` + scratchDB)
	require.NoError(t, err)

	m, err := migrate.New("file://migrations", dsn(scratchDB))
	require.NoError(t, err)
	t.Cleanup(func() { m.Close() })
	require.NoError(t, m.Migrate(version))

	db, err := sql.Open("postgres", dsn(scratchDB))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, m
}

func up(t *testing.T, m *migrate.Migrate) {
	t.Helper()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migrate up: %v", err)
	}
}

// Scenario 8 (dates): full dates entered before partial and qualified dates
// existed keep their values. A life event date becomes exact, since nothing
// says it was anything else, and recorded_date becomes registration_date.
func TestMigration_DatesCarriedForward(t *testing.T) {
	db, m := migrateTo(t, 10)

	var caseID, personID, eventID, undatedID, docID string
	require.NoError(t, db.QueryRow(`INSERT INTO cases (title) VALUES ('Rossi') RETURNING id`).Scan(&caseID))
	require.NoError(t, db.QueryRow(
		`INSERT INTO people (case_id, first_name, last_name) VALUES ($1, 'Giuseppe', 'Rossi') RETURNING id`,
		caseID).Scan(&personID))
	require.NoError(t, db.QueryRow(`
		INSERT INTO life_events (case_id, person_id, event_type, event_date, spouse_birth_date)
		VALUES ($1, $2, 'marriage', '1902-06-10', '1880-01-31') RETURNING id`,
		caseID, personID).Scan(&eventID))
	require.NoError(t, db.QueryRow(`
		INSERT INTO life_events (case_id, person_id, event_type) VALUES ($1, $2, 'other') RETURNING id`,
		caseID, personID).Scan(&undatedID))
	require.NoError(t, db.QueryRow(`
		INSERT INTO documents
			(case_id, person_id, life_event_id, document_type, title,
			 issue_date, recorded_date, recorded_birth_date,
			 official_copy_status_id, amendment_status_id, apostille_status_id, translation_status_id)
		VALUES ($1, $2, $3, 'marriage_certificate', 'Marriage certificate',
			'1990-05-04', '1902-06-12', '0890-01-01',
			(SELECT id FROM document_statuses WHERE system_key = 'official_copy_default'),
			(SELECT id FROM document_statuses WHERE system_key = 'amendment_default'),
			(SELECT id FROM document_statuses WHERE system_key = 'apostille_default'),
			(SELECT id FROM document_statuses WHERE system_key = 'translation_default'))
		RETURNING id`,
		caseID, personID, eventID).Scan(&docID))

	up(t, m)

	var date, qualifier, spouseDate, spouseQualifier string
	var end, spouseEnd sql.NullString
	require.NoError(t, db.QueryRow(`
		SELECT event_date, event_date_qualifier, event_date_end,
		       spouse_birth_date, spouse_birth_date_qualifier, spouse_birth_date_end
		FROM life_events WHERE id = $1`, eventID).
		Scan(&date, &qualifier, &end, &spouseDate, &spouseQualifier, &spouseEnd))
	assert.Equal(t, "1902-06-10", date)
	assert.Equal(t, "exact", qualifier)
	assert.False(t, end.Valid)
	assert.Equal(t, "1880-01-31", spouseDate)
	assert.Equal(t, "exact", spouseQualifier)
	assert.False(t, spouseEnd.Valid)

	var undatedDate, undatedQualifier sql.NullString
	require.NoError(t, db.QueryRow(
		`SELECT event_date, event_date_qualifier FROM life_events WHERE id = $1`, undatedID).
		Scan(&undatedDate, &undatedQualifier))
	assert.False(t, undatedDate.Valid, "an unknown date stays unknown")
	assert.False(t, undatedQualifier.Valid, "an unknown date has no qualifier")

	var issue, registration, recordedBirth string
	require.NoError(t, db.QueryRow(
		`SELECT issue_date, registration_date, recorded_birth_date FROM documents WHERE id = $1`, docID).
		Scan(&issue, &registration, &recordedBirth))
	assert.Equal(t, "1990-05-04", issue)
	assert.Equal(t, "1902-06-12", registration)
	assert.Equal(t, "0890-01-01", recordedBirth, "early years keep four digits")
}

// Scenario 13 (birth and death): a person's own birth and death fields move
// to their birth and death life events. An event's own value is kept when the
// two disagree, and the person's goes into the event's notes, so nothing is
// lost. The person's columns are then removed.
func TestMigration_BirthAndDeathMoveToEvents(t *testing.T) {
	db, m := migrateTo(t, 11)

	var caseID string
	require.NoError(t, db.QueryRow(`INSERT INTO cases (title) VALUES ('Rossi') RETURNING id`).Scan(&caseID))
	person := func(name, birthDate, birthPlace, deathDate any) string {
		var id string
		require.NoError(t, db.QueryRow(`
			INSERT INTO people (case_id, first_name, last_name, birth_date, birth_place, death_date)
			VALUES ($1, $2, 'Rossi', $3, $4, $5) RETURNING id`,
			caseID, name, birthDate, birthPlace, deathDate).Scan(&id))
		return id
	}
	event := func(personID, eventType string, date, qualifier, place, notes any) {
		_, err := db.Exec(`
			INSERT INTO life_events (case_id, person_id, event_type, event_date, event_date_qualifier, event_place, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			caseID, personID, eventType, date, qualifier, place, notes)
		require.NoError(t, err)
	}

	noEvent := person("NoEvent", "1875-03-15", "Palermo", "1952-11-03")
	emptyEvent := person("EmptyEvent", "1880-01-31", "Naples", nil)
	event(emptyEvent, "birth", nil, nil, nil, "Baptised a week later")
	disagree := person("Disagree", "1890-03-15", "Rome", nil)
	event(disagree, "birth", "1890", "about", "Roma", nil)
	nothing := person("Nothing", nil, nil, nil)

	up(t, m)

	type ev struct {
		date, qualifier, place, notes sql.NullString
	}
	get := func(personID, eventType string) []ev {
		rows, err := db.Query(`
			SELECT event_date, event_date_qualifier::text, event_place, notes
			FROM life_events WHERE person_id = $1 AND event_type = $2 AND deleted_at IS NULL`, personID, eventType)
		require.NoError(t, err)
		defer rows.Close()
		var out []ev
		for rows.Next() {
			var e ev
			require.NoError(t, rows.Scan(&e.date, &e.qualifier, &e.place, &e.notes))
			out = append(out, e)
		}
		return out
	}
	str := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

	assert.Equal(t, []ev{{date: str("1875-03-15"), qualifier: str("exact"), place: str("Palermo")}},
		get(noEvent, "birth"), "a birth event is created")
	assert.Equal(t, []ev{{date: str("1952-11-03"), qualifier: str("exact")}},
		get(noEvent, "death"), "a death event is created")
	assert.Equal(t, []ev{{date: str("1880-01-31"), qualifier: str("exact"), place: str("Naples"), notes: str("Baptised a week later")}},
		get(emptyEvent, "birth"), "the existing event's empty fields are filled")
	assert.Equal(t, []ev{{date: str("1890"), qualifier: str("about"), place: str("Roma"),
		notes: str("Birth date on the person record: 1890-03-15\nBirth place on the person record: Rome")}},
		get(disagree, "birth"), "the event keeps its own values; the person's go into its notes")
	assert.Empty(t, get(nothing, "birth"), "no fields, so no event")
	assert.Empty(t, get(nothing, "death"))

	var columns int
	require.NoError(t, db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'people' AND column_name IN ('birth_date', 'birth_place', 'death_date')`).Scan(&columns))
	assert.Zero(t, columns, "the person's own columns are removed")
}

// Scenario 13, continued: a person with two active birth events cannot be
// migrated, because which one is theirs is a judgement. The migration stops
// and names them.
func TestMigration_TwoBirthEventsStop(t *testing.T) {
	db, m := migrateTo(t, 11)

	var caseID, personID string
	require.NoError(t, db.QueryRow(`INSERT INTO cases (title) VALUES ('Rossi') RETURNING id`).Scan(&caseID))
	require.NoError(t, db.QueryRow(
		`INSERT INTO people (case_id, first_name, last_name) VALUES ($1, 'Giuseppe', 'Rossi') RETURNING id`,
		caseID).Scan(&personID))
	for range 2 {
		_, err := db.Exec(`INSERT INTO life_events (case_id, person_id, event_type) VALUES ($1, $2, 'birth')`, caseID, personID)
		require.NoError(t, err)
	}

	err := m.Up()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Giuseppe Rossi")
}

// Scenario 12 (marriages): spouse details already saved on events that are not
// marriages move into those events' notes, so nothing is lost, and marriages
// keep theirs. Afterwards the database refuses spouse or divorce details on
// any other event, and a linked spouse together with flat spouse details.
func TestMigration_SpouseDetailsOnlyOnMarriages(t *testing.T) {
	db, m := migrateTo(t, 12)

	var caseID, personID, spouseID string
	require.NoError(t, db.QueryRow(`INSERT INTO cases (title) VALUES ('Rossi') RETURNING id`).Scan(&caseID))
	require.NoError(t, db.QueryRow(
		`INSERT INTO people (case_id, first_name, last_name) VALUES ($1, 'Giuseppe', 'Rossi') RETURNING id`,
		caseID).Scan(&personID))
	require.NoError(t, db.QueryRow(
		`INSERT INTO people (case_id, first_name, last_name) VALUES ($1, 'Maria', 'Bianchi') RETURNING id`,
		caseID).Scan(&spouseID))
	event := func(eventType string, notes any) string {
		var id string
		require.NoError(t, db.QueryRow(`
			INSERT INTO life_events (case_id, person_id, event_type, spouse_name,
				spouse_birth_date, spouse_birth_date_qualifier, spouse_birth_date_end, spouse_birth_place, notes)
			VALUES ($1, $2, $3, 'Maria Bianchi', '1880', 'between', '1882', 'Palermo', $4) RETURNING id`,
			caseID, personID, eventType, notes).Scan(&id))
		return id
	}
	death := event("death", "Buried in Brooklyn")
	marriage := event("marriage", nil)

	up(t, m)

	var name, place, notes sql.NullString
	require.NoError(t, db.QueryRow(
		`SELECT spouse_name, spouse_birth_place, notes FROM life_events WHERE id = $1`, death,
	).Scan(&name, &place, &notes))
	assert.False(t, name.Valid, "spouse details are cleared from the death event")
	assert.False(t, place.Valid)
	assert.Equal(t, "Buried in Brooklyn\nSpouse name: Maria Bianchi\nSpouse birth date: between 1880 and 1882\nSpouse birth place: Palermo",
		notes.String, "and kept in its notes")

	require.NoError(t, db.QueryRow(
		`SELECT spouse_name, spouse_birth_place, notes FROM life_events WHERE id = $1`, marriage,
	).Scan(&name, &place, &notes))
	assert.Equal(t, "Maria Bianchi", name.String, "a marriage keeps its spouse details")
	assert.Equal(t, "Palermo", place.String)
	assert.False(t, notes.Valid)

	refused := map[string]string{
		"divorce details on another event": `INSERT INTO life_events (case_id, person_id, event_type, divorce_place)
			VALUES ($1, $2, 'other', 'Reno')`,
		"a linked spouse on another event": `INSERT INTO life_events (case_id, person_id, event_type, spouse_person_id)
			VALUES ($1, $2, 'other', $3)`,
		"a linked spouse with a flat name": `INSERT INTO life_events (case_id, person_id, event_type, spouse_person_id, spouse_name)
			VALUES ($1, $2, 'marriage', $3, 'Maria')`,
		"the event's own person as spouse": `INSERT INTO life_events (case_id, person_id, event_type, spouse_person_id)
			VALUES ($1, $2, 'marriage', $2)`,
	}
	_, err := db.Exec(`INSERT INTO life_events (case_id, person_id, event_type, spouse_person_id)
		VALUES ($1, $2, 'marriage', $3)`, caseID, personID, spouseID)
	require.NoError(t, err, "a linked spouse on a marriage is accepted, so the refusals below are the checks at work")
	for what, query := range refused {
		args := []any{caseID, personID}
		if strings.Contains(query, "$3") {
			args = append(args, spouseID)
		}
		_, err := db.Exec(query, args...)
		assert.Error(t, err, what)
	}
}

// insertDocument adds a document in the shape before document tracking, with
// its phases at their defaults and the given check and update time.
func insertDocument(t *testing.T, db *sql.DB, caseID, personID string, verified bool, verifiedAt, updatedAt any) string {
	t.Helper()
	var id string
	require.NoError(t, db.QueryRow(`
		INSERT INTO documents
			(case_id, person_id, document_type, title, is_verified, verified_at, updated_at,
			 official_copy_status_id, amendment_status_id, apostille_status_id, translation_status_id)
		VALUES ($1, $2, 'birth_certificate', 'Birth certificate', $3, $4, $5,
			(SELECT id FROM document_statuses WHERE system_key = 'official_copy_default'),
			(SELECT id FROM document_statuses WHERE system_key = 'amendment_default'),
			(SELECT id FROM document_statuses WHERE system_key = 'apostille_default'),
			(SELECT id FROM document_statuses WHERE system_key = 'translation_default'))
		RETURNING id`,
		caseID, personID, verified, verifiedAt, updatedAt).Scan(&id))
	return id
}

func casePerson(t *testing.T, db *sql.DB) (caseID, personID string) {
	t.Helper()
	require.NoError(t, db.QueryRow(`INSERT INTO cases (title) VALUES ('Rossi') RETURNING id`).Scan(&caseID))
	require.NoError(t, db.QueryRow(
		`INSERT INTO people (case_id, first_name, last_name) VALUES ($1, 'Giuseppe', 'Rossi') RETURNING id`,
		caseID).Scan(&personID))
	return caseID, personID
}

// Scenario 9 (document tracking): a verified document stays checked, at the
// time it was verified, with no one recorded as having checked it.
func TestMigration_VerifiedBecomesDetailsChecked(t *testing.T) {
	db, m := migrateTo(t, 13)
	caseID, personID := casePerson(t, db)
	verifiedAt := time.Date(2026, 3, 2, 14, 30, 0, 0, time.UTC)
	verified := insertDocument(t, db, caseID, personID, true, verifiedAt, verifiedAt)
	unverified := insertDocument(t, db, caseID, personID, false, nil, verifiedAt)

	up(t, m)

	var checked bool
	var at sql.NullTime
	var by sql.NullString
	require.NoError(t, db.QueryRow(
		`SELECT details_checked, details_checked_at, details_checked_by FROM documents WHERE id = $1`, verified,
	).Scan(&checked, &at, &by))
	assert.True(t, checked)
	assert.True(t, verifiedAt.Equal(at.Time), "checked at the time it was verified")
	assert.False(t, by.Valid, "no one is recorded as having checked it")

	require.NoError(t, db.QueryRow(
		`SELECT details_checked, details_checked_at, details_checked_by FROM documents WHERE id = $1`, unverified,
	).Scan(&checked, &at, &by))
	assert.False(t, checked)
	assert.False(t, at.Valid)
	assert.False(t, by.Valid)
}

// Scenario 17 (document tracking): existing documents' phases are dated from
// the document's last update, the best time known.
func TestMigration_PhasesSinceLastUpdate(t *testing.T) {
	db, m := migrateTo(t, 13)
	caseID, personID := casePerson(t, db)
	updatedAt := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	doc := insertDocument(t, db, caseID, personID, false, nil, updatedAt)

	up(t, m)

	for _, phase := range []string{"official_copy", "amendment", "apostille", "translation"} {
		var since time.Time
		var reference sql.NullString
		require.NoError(t, db.QueryRow(
			`SELECT `+phase+`_status_since, `+phase+`_reference FROM documents WHERE id = $1`, doc,
		).Scan(&since, &reference))
		assert.True(t, updatedAt.Equal(since), "%s since is the last update", phase)
		assert.False(t, reference.Valid, phase)
	}
	var forSubmission bool
	require.NoError(t, db.QueryRow(`SELECT for_submission FROM documents WHERE id = $1`, doc).Scan(&forSubmission))
	assert.True(t, forSubmission, "existing documents are for submission")
}
