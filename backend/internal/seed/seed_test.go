// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package seed_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/seed"
	"github.com/tfior/doc-tracker/internal/testhelpers"
	"github.com/tfior/doc-tracker/internal/users"
)

// fresh removes any seed cases left by an earlier run, so the seed starts
// from nothing. Everything in a case goes with it.
func fresh(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`DELETE FROM cases WHERE title = ANY($1)`, pq.Array(seed.CaseTitles))
	require.NoError(t, err)
}

func caseID(t *testing.T, db *sql.DB, title string) string {
	t.Helper()
	var id string
	require.NoError(t, db.QueryRow(`SELECT id::text FROM cases WHERE title = $1`, title).Scan(&id), title)
	return id
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(query, args...).Scan(&n), query)
	return n
}

// Scenario 1
func TestSeed_CreatesBothCases(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	fresh(t, db)
	require.NoError(t, seed.Run(context.Background(), db))

	rossi := caseID(t, db, "Rossi → Martini")
	assert.Equal(t, 4, count(t, db, `SELECT COUNT(*) FROM people WHERE case_id = $1`, rossi))
	assert.Equal(t, 3, count(t, db, `SELECT COUNT(*) FROM person_relationships WHERE case_id = $1`, rossi))
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM claim_lines WHERE case_id = $1 AND status = 'ineligible'`, rossi))
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM claim_lines WHERE case_id = $1`, rossi))
	assert.Positive(t, count(t, db, `SELECT COUNT(*) FROM life_events WHERE case_id = $1`, rossi))
	assert.Positive(t, count(t, db, `SELECT COUNT(*) FROM documents WHERE case_id = $1`, rossi))

	esposito := caseID(t, db, "Esposito → Keller")
	assert.Equal(t, 4, count(t, db, `SELECT COUNT(*) FROM people WHERE case_id = $1`, esposito))
	assert.Equal(t, 3, count(t, db, `SELECT COUNT(*) FROM person_relationships WHERE case_id = $1`, esposito))
	statuses := map[string]string{}
	rows, err := db.Query(`
		SELECT p.first_name, cl.status::text FROM claim_lines cl JOIN people p ON p.id = cl.root_person_id
		WHERE cl.case_id = $1`, esposito)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var name, status string
		require.NoError(t, rows.Scan(&name, &status))
		statuses[name] = status
	}
	assert.Equal(t, map[string]string{"Paolo": "ineligible", "Lucia": "eligible"}, statuses)
	assert.Equal(t, 1, count(t, db, `
		SELECT COUNT(*) FROM life_events le
		JOIN people p ON p.id = le.person_id JOIN people s ON s.id = le.spouse_person_id
		WHERE le.case_id = $1 AND le.event_type = 'marriage' AND p.first_name = 'Paolo' AND s.first_name = 'Lucia'`,
		esposito), "the grandparents share one marriage")
}

// Scenario 2
func TestSeed_EverythingLoggedToTheLockedSeedUser(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	fresh(t, db)
	require.NoError(t, seed.Run(context.Background(), db))

	u, err := users.NewService(users.NewStore(db)).GetByEmail(context.Background(), seed.UserEmail)
	require.NoError(t, err)
	assert.Equal(t, "Seed", u.FirstName)
	assert.Equal(t, "Data", u.LastName)

	ids := []string{caseID(t, db, "Rossi → Martini"), caseID(t, db, "Esposito → Keller")}
	assert.Zero(t, count(t, db, `
		SELECT COUNT(*) FROM activity_logs WHERE case_id = ANY($1::uuid[]) AND user_id <> $2`,
		pq.Array(ids), u.ID), "every entry is credited to the seed user")
	for table, entity := range map[string]string{
		"cases": "case", "people": "person", "life_events": "life_event",
		"documents": "document", "claim_lines": "claim_line",
	} {
		caseColumn := "case_id"
		if table == "cases" {
			caseColumn = "id"
		}
		assert.Zero(t, count(t, db, `
			SELECT COUNT(*) FROM `+table+` r
			WHERE r.`+caseColumn+` = ANY($1::uuid[])
			  AND NOT EXISTS (SELECT 1 FROM activity_logs a
			                  WHERE a.entity_type = $2 AND a.entity_id = r.id AND a.action = 'created')`,
			pq.Array(ids), entity), "every %s has a created entry", entity)
	}

	assert.Equal(t, users.LockedPasswordHash, u.PasswordHash, "the account is locked after seeding")
	c := testhelpers.NewAnonymousClient(db)
	for _, password := range []string{"!", "", "password"} {
		rec := c.Do("POST", "/api/v1/auth/session", map[string]string{"email": seed.UserEmail, "password": password})
		assert.NotEqual(t, http.StatusOK, rec.Code, "logging in with %q must fail", password)
		assert.Empty(t, rec.Result().Cookies(), "no session for %q", password)
	}
}

// Scenario 3
func TestSeed_SecondRunChangesNothing(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	fresh(t, db)
	require.NoError(t, seed.Run(context.Background(), db))

	snapshot := func() []int {
		ids := pq.Array([]string{caseID(t, db, "Rossi → Martini"), caseID(t, db, "Esposito → Keller")})
		var out []int
		for _, table := range []string{"people", "life_events", "documents", "claim_lines", "activity_logs"} {
			out = append(out, count(t, db, `SELECT COUNT(*) FROM `+table+` WHERE case_id = ANY($1::uuid[])`, ids))
		}
		return append(out, count(t, db, `SELECT COUNT(*) FROM cases WHERE title = ANY($1)`, pq.Array(seed.CaseTitles)))
	}
	before := snapshot()

	require.NoError(t, seed.Run(context.Background(), db))

	assert.Equal(t, before, snapshot())
}
