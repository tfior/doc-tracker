// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package testhelpers

import (
	"database/sql"
	"testing"
)

// LogEntry is an activity log row as seen by tests.
type LogEntry struct {
	Action     string
	EntityType string
	EntityID   string
	UserID     string
}

// ActivityLog returns every activity log entry for a case, oldest first.
func ActivityLog(t *testing.T, db *sql.DB, caseID string) []LogEntry {
	t.Helper()
	rows, err := db.Query(`
		SELECT action::text, entity_type::text, entity_id::text, user_id::text
		FROM activity_logs WHERE case_id = $1
		ORDER BY created_at, id`, caseID)
	if err != nil {
		t.Fatalf("query activity log: %v", err)
	}
	defer rows.Close()
	entries := []LogEntry{}
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.Action, &e.EntityType, &e.EntityID, &e.UserID); err != nil {
			t.Fatalf("scan activity log: %v", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate activity log: %v", err)
	}
	return entries
}

// FailActivityLogWrites makes every insert into activity_logs fail until the
// test ends, by installing a trigger in the test database. Nothing in the
// application is faked: the failure happens where a real one would.
func FailActivityLogWrites(t *testing.T, db *sql.DB) {
	t.Helper()
	drop := func() {
		_, _ = db.Exec(`DROP TRIGGER IF EXISTS test_reject_activity_log ON activity_logs`)
		_, _ = db.Exec(`DROP FUNCTION IF EXISTS test_reject_activity_log()`)
	}
	drop()
	if _, err := db.Exec(`
		CREATE FUNCTION test_reject_activity_log() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'activity log writes disabled by test';
		END $$ LANGUAGE plpgsql`); err != nil {
		t.Fatalf("create rejecting function: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TRIGGER test_reject_activity_log
		BEFORE INSERT ON activity_logs
		FOR EACH ROW EXECUTE FUNCTION test_reject_activity_log()`); err != nil {
		t.Fatalf("create rejecting trigger: %v", err)
	}
	t.Cleanup(drop)
}
