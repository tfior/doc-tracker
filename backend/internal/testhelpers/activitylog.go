// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package testhelpers

import (
	"database/sql"
	"encoding/json"
	"testing"
)

// LogEntry is an activity log row as seen by tests.
type LogEntry struct {
	Action     string
	EntityType string
	EntityID   string
	EntityName string
	UserID     string
	Changes    []Change
}

// Change is one field change in a log entry. From and To are nil for null.
type Change struct {
	Field string `json:"field"`
	From  any    `json:"from"`
	To    any    `json:"to"`
}

// ActivityLog returns every activity log entry for a case, oldest first.
func ActivityLog(t *testing.T, db *sql.DB, caseID string) []LogEntry {
	t.Helper()
	rows, err := db.Query(`
		SELECT action::text, entity_type::text, entity_id::text, entity_name, user_id::text, changes
		FROM activity_logs WHERE case_id = $1
		ORDER BY created_at, id`, caseID)
	if err != nil {
		t.Fatalf("query activity log: %v", err)
	}
	defer rows.Close()
	entries := []LogEntry{}
	for rows.Next() {
		var e LogEntry
		var changes []byte
		if err := rows.Scan(&e.Action, &e.EntityType, &e.EntityID, &e.EntityName, &e.UserID, &changes); err != nil {
			t.Fatalf("scan activity log: %v", err)
		}
		if changes != nil {
			if err := json.Unmarshal(changes, &e.Changes); err != nil {
				t.Fatalf("decode activity log changes: %v", err)
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate activity log: %v", err)
	}
	return entries
}

// FailActivityLogWrites makes every insert into activity_logs fail until the
// test ends. See FailInserts.
func FailActivityLogWrites(t *testing.T, db *sql.DB) {
	t.Helper()
	FailInserts(t, db, "activity_logs")
}

// FailInserts makes every insert into a table fail until the test ends, by
// installing a trigger in the test database. Nothing in the application is
// faked: the failure happens where a real one would.
func FailInserts(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	trigger := "test_reject_" + table
	drop := func() {
		_, _ = db.Exec(`DROP TRIGGER IF EXISTS ` + trigger + ` ON ` + table)
		_, _ = db.Exec(`DROP FUNCTION IF EXISTS test_reject_insert() CASCADE`)
	}
	drop()
	if _, err := db.Exec(`
		CREATE OR REPLACE FUNCTION test_reject_insert() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'inserts disabled by test';
		END $$ LANGUAGE plpgsql`); err != nil {
		t.Fatalf("create rejecting function: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TRIGGER ` + trigger + `
		BEFORE INSERT ON ` + table + `
		FOR EACH ROW EXECUTE FUNCTION test_reject_insert()`); err != nil {
		t.Fatalf("create rejecting trigger: %v", err)
	}
	t.Cleanup(drop)
}
