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
	"testing"

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
