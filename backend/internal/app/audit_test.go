// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package app_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/testhelpers"
)

// These tests cover the activity log's content (domain model, Activity Log):
// what changed, from what, to what; readable names; restores and permanent
// deletes; and the record that survives a case's permanent deletion.

// lastEntry returns the newest log entry for an entity, failing if there is none.
func (e *env) lastEntry(t *testing.T, caseID, entityType, entityID string) testhelpers.LogEntry {
	t.Helper()
	var found *testhelpers.LogEntry
	for _, entry := range testhelpers.ActivityLog(t, e.db, caseID) {
		if entry.EntityType == entityType && entry.EntityID == entityID {
			entry := entry
			found = &entry
		}
	}
	require.NotNil(t, found, "no log entry for %s %s", entityType, entityID)
	return *found
}

func (e *env) entryCount(t *testing.T, caseID string) int {
	t.Helper()
	return len(testhelpers.ActivityLog(t, e.db, caseID))
}

func (e *env) statusID(t *testing.T, phase, label string) string {
	t.Helper()
	rec := e.c.Do("GET", "/api/v1/document-statuses?phase="+phase, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var statuses []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&statuses))
	for _, s := range statuses {
		if s.Label == label {
			return s.ID
		}
	}
	t.Fatalf("no %q status for phase %s", label, phase)
	return ""
}

// ---------------------------------------------------------------------------
// A. What changed, from what, to what
// ---------------------------------------------------------------------------

// Scenario 1
func TestLog_UpdateRecordsOnlyFieldsThatChanged(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/people/"+p,
		map[string]any{"first_name": "Joseph", "last_name": "Rossi"}) // last name unchanged
	require.Equal(t, http.StatusOK, rec.Code)

	entry := e.lastEntry(t, c, "person", p)
	assert.Equal(t, "updated", entry.Action)
	assert.Equal(t, []testhelpers.Change{{Field: "first_name", From: "Giuseppe", To: "Joseph"}}, entry.Changes)
}

// Scenario 2
func TestLog_SettingAndClearingRecordNull(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	path := "/api/v1/cases/" + c + "/people/" + p

	require.Equal(t, http.StatusOK, e.c.Do("PATCH", path, map[string]any{"notes": "Emigrated 1905"}).Code)
	assert.Equal(t, []testhelpers.Change{{Field: "notes", From: nil, To: "Emigrated 1905"}},
		e.lastEntry(t, c, "person", p).Changes)

	require.Equal(t, http.StatusOK, e.c.Do("PATCH", path, map[string]any{"notes": nil}).Code)
	assert.Equal(t, []testhelpers.Change{{Field: "notes", From: "Emigrated 1905", To: nil}},
		e.lastEntry(t, c, "person", p).Changes)
}

// Scenario 3
func TestLog_SaveThatChangesNothingWritesNothing(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	before := e.entryCount(t, c)

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/people/"+p,
		map[string]any{"first_name": "Giuseppe", "last_name": "Rossi", "notes": nil})
	require.Equal(t, http.StatusOK, rec.Code)

	assert.Equal(t, before, e.entryCount(t, c))
}

// Scenario 4
func TestLog_StatusChangeRecordsLabels(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	doc := e.createDocument(t, c, p, "")

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc+"/status",
		map[string]any{"phase": "official_copy", "status_id": e.statusID(t, "official_copy", "Requested")})
	require.Equal(t, http.StatusOK, rec.Code)

	assert.Equal(t, []testhelpers.Change{{Field: "official_copy_status", From: "Not Started", To: "Requested"}},
		e.lastEntry(t, c, "document", doc).Changes)
}

// Scenario 5
func TestLog_ReassignRecordsPersonAndEvent(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	a, b := e.createPerson(t, c, "Antonio"), e.createPerson(t, c, "Carlo")
	evA, evB := e.createLifeEvent(t, c, a), e.createLifeEvent(t, c, b)
	doc := e.createDocument(t, c, a, evA)

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc+"/parent",
		map[string]any{"person_id": b, "life_event_id": evB})
	require.Equal(t, http.StatusOK, rec.Code)

	assert.ElementsMatch(t, []testhelpers.Change{
		{Field: "person_id", From: a, To: b},
		{Field: "life_event_id", From: evA, To: evB},
	}, e.lastEntry(t, c, "document", doc).Changes)
}

// ---------------------------------------------------------------------------
// B. Readable names
// ---------------------------------------------------------------------------

// Scenario 6
func TestLog_EveryEntryHasReadableName(t *testing.T) {
	e := newEnv(t)
	c := e.c.MustCreate(t, "/api/v1/cases", map[string]any{"title": "Rossi family"})
	giuseppe := e.createPerson(t, c, "Giuseppe")
	antonio := e.createPerson(t, c, "Antonio")
	e.mustAddParent(t, c, antonio, giuseppe)
	ev := e.createLifeEvent(t, c, giuseppe)
	doc := e.c.MustCreate(t, "/api/v1/cases/"+c+"/documents", map[string]any{
		"person_id": giuseppe, "life_event_id": ev, "document_type": "birth_certificate", "title": "Italian birth certificate",
	})
	line := e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": giuseppe})

	// Delete everything again, so delete entries are checked too.
	e.trash(t, "/api/v1/cases/"+c+"/claim-lines/"+line)
	e.trash(t, "/api/v1/cases/"+c+"/documents/"+doc)
	e.trash(t, "/api/v1/cases/"+c+"/life-events/"+ev)
	require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", "/api/v1/cases/"+c+"/people/"+antonio+"/relationships/"+giuseppe, nil).Code)
	e.trash(t, "/api/v1/cases/"+c+"/people/"+antonio)
	e.trash(t, "/api/v1/cases/"+c)

	want := map[string]string{
		"case":                "Rossi family",
		"person":              "", // checked per person below
		"person_relationship": "Giuseppe Rossi as parent of Antonio Rossi",
		"life_event":          "Birth life event for Giuseppe Rossi",
		"document":            "Italian birth certificate",
		"claim_line":          "Claim line through Giuseppe Rossi",
	}
	people := map[string]string{giuseppe: "Giuseppe Rossi", antonio: "Antonio Rossi"}
	seen := map[string]bool{}
	for _, entry := range testhelpers.ActivityLog(t, e.db, c) {
		seen[entry.EntityType+" "+entry.Action] = true
		if entry.EntityType == "person" {
			assert.Equal(t, people[entry.EntityID], entry.EntityName, "person %s", entry.Action)
			continue
		}
		assert.Equal(t, want[entry.EntityType], entry.EntityName, "%s %s", entry.EntityType, entry.Action)
	}
	for _, k := range []string{"case created", "case deleted", "person deleted", "person_relationship created",
		"person_relationship deleted", "life_event deleted", "document deleted", "claim_line deleted"} {
		assert.True(t, seen[k], "no %q entry", k)
	}
}

// Scenario 7
func TestLog_NameCapturedAtTheTime(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	require.Equal(t, http.StatusOK, e.c.Do("PATCH", "/api/v1/cases/"+c+"/people/"+p, map[string]any{"first_name": "Joseph"}).Code)

	entries := testhelpers.ActivityLog(t, e.db, c)
	require.GreaterOrEqual(t, len(entries), 2)
	var created testhelpers.LogEntry
	for _, entry := range entries {
		if entry.EntityID == p && entry.Action == "created" {
			created = entry
		}
	}
	assert.Equal(t, "Giuseppe Rossi", created.EntityName, "the created entry was renamed")
	assert.Equal(t, "Joseph Rossi", e.lastEntry(t, c, "person", p).EntityName)
}

// ---------------------------------------------------------------------------
// C. Restore and permanent delete
// ---------------------------------------------------------------------------

// Scenarios 8 and 9
func TestLog_RestoreAndPermanentDelete(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	ev := e.createLifeEvent(t, c, p)
	doc := e.createDocument(t, c, p, ev)
	line := e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": p})

	items := []struct{ entityType, path string }{
		{"claim_line", "/api/v1/cases/" + c + "/claim-lines/" + line},
		{"document", "/api/v1/cases/" + c + "/documents/" + doc},
		{"life_event", "/api/v1/cases/" + c + "/life-events/" + ev},
		{"person", "/api/v1/cases/" + c + "/people/" + p},
	}
	ids := map[string]string{"claim_line": line, "document": doc, "life_event": ev, "person": p}

	for _, it := range items { // trash and restore each
		e.trash(t, it.path)
		require.Equal(t, http.StatusNoContent, e.c.Do("POST", it.path+"/restore", nil).Code, it.entityType)
		assert.Equal(t, "restored", e.lastEntry(t, c, it.entityType, ids[it.entityType]).Action, it.entityType)
	}
	for _, it := range items { // then trash and permanently delete each, children first
		e.trash(t, it.path)
		require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", it.path+"/permanent", nil).Code, it.entityType)
		entry := e.lastEntry(t, c, it.entityType, ids[it.entityType])
		assert.Equal(t, "permanently_deleted", entry.Action, it.entityType)
		assert.NotEmpty(t, entry.EntityName, it.entityType)
	}

	t.Run("case restore", func(t *testing.T) {
		c2 := e.createCase(t)
		e.trash(t, "/api/v1/cases/"+c2)
		require.Equal(t, http.StatusNoContent, e.c.Do("POST", "/api/v1/cases/"+c2+"/restore", nil).Code)
		assert.Equal(t, "restored", e.lastEntry(t, c2, "case", c2).Action)
	})
}

// Scenario 10
func TestLog_RefusedRestoreWritesNothing(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	doc := e.createDocument(t, c, p, "")
	e.trash(t, "/api/v1/cases/"+c+"/documents/"+doc)
	e.trash(t, "/api/v1/cases/"+c+"/people/"+p)
	before := e.entryCount(t, c)

	require.Equal(t, http.StatusConflict, e.c.Do("POST", "/api/v1/cases/"+c+"/documents/"+doc+"/restore", nil).Code)

	assert.Equal(t, before, e.entryCount(t, c))
}

// ---------------------------------------------------------------------------
// D. The case deletion record
// ---------------------------------------------------------------------------

type deletionRecord struct {
	caseTitle                                string
	deletedBy                                sql.NullString
	deletedAt                                time.Time
	people, lifeEvents, documents, fileCount int
}

func readDeletionRecord(t *testing.T, db *sql.DB, caseID string) (deletionRecord, bool) {
	t.Helper()
	var r deletionRecord
	err := db.QueryRow(`
		SELECT case_title, deleted_by::text, deleted_at, people_count, life_events_count, documents_count, files_count
		FROM case_deletion_records WHERE case_id = $1`, caseID,
	).Scan(&r.caseTitle, &r.deletedBy, &r.deletedAt, &r.people, &r.lifeEvents, &r.documents, &r.fileCount)
	if err == sql.ErrNoRows {
		return r, false
	}
	require.NoError(t, err)
	return r, true
}

// Scenarios 11 and 12
func TestPermanentDeleteCase_WritesDeletionRecord(t *testing.T) {
	e := newEnv(t)
	c := e.c.MustCreate(t, "/api/v1/cases", map[string]any{"title": "Rossi family"})
	p1, p2 := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Antonio")
	ev := e.createLifeEvent(t, c, p1)
	doc := e.createDocument(t, c, p1, ev)
	e.createDocument(t, c, p2, "")
	e.trash(t, "/api/v1/cases/"+c+"/people/"+p2) // trashed records count too
	// File attachments have no API until Milestone 8, so one is inserted directly.
	_, err := e.db.Exec(`INSERT INTO file_attachments (document_id, storage_key, filename, content_type, size_bytes, attachment_type)
		VALUES ($1, 'k', 'scan.pdf', 'application/pdf', 1, 'original')`, doc)
	require.NoError(t, err)
	e.trash(t, "/api/v1/cases/"+c)
	started := time.Now()

	require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", "/api/v1/cases/"+c+"/permanent", nil).Code)

	r, ok := readDeletionRecord(t, e.db, c)
	require.True(t, ok, "no deletion record was written")
	assert.Equal(t, "Rossi family", r.caseTitle)
	assert.Equal(t, e.c.UserID, r.deletedBy.String)
	assert.WithinDuration(t, started, r.deletedAt, time.Minute)
	assert.Equal(t, 2, r.people)
	assert.Equal(t, 1, r.lifeEvents)
	assert.Equal(t, 2, r.documents)
	assert.Equal(t, 1, r.fileCount)
	assert.Empty(t, testhelpers.ActivityLog(t, e.db, c), "the case's activity log should go with it")
}

// Scenario 13
func TestPermanentDeleteCase_RecordFails_CaseNotDeleted(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	e.trash(t, "/api/v1/cases/"+c)
	testhelpers.FailInserts(t, e.db, "case_deletion_records")

	rec := e.c.Do("DELETE", "/api/v1/cases/"+c+"/permanent", nil)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var stillThere int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM cases WHERE id = $1`, c).Scan(&stillThere))
	assert.Equal(t, 1, stillThere, "the case was deleted although its deletion record failed")
}

// permanentlyDeleted returns the IDs logged as permanently deleted for an entity type.
func (e *env) permanentlyDeleted(t *testing.T, caseID, entityType string) []string {
	t.Helper()
	out := []string{}
	for _, entry := range testhelpers.ActivityLog(t, e.db, caseID) {
		if entry.EntityType == entityType && entry.Action == "permanently_deleted" {
			assert.NotEmpty(t, entry.EntityName, "%s %s has no name", entityType, entry.EntityID)
			out = append(out, entry.EntityID)
		}
	}
	return out
}

// Scenario 14
func TestLog_PermanentDeletePerson_LogsEverythingThatGoesWithThem(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	parent, p, child := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Antonio"), e.createPerson(t, c, "Carlo")
	e.mustAddParent(t, c, p, parent) // a link where p is the child
	e.mustAddParent(t, c, child, p)  // a link where p is the parent
	ev1, ev2 := e.createLifeEvent(t, c, p), e.createLifeEvent(t, c, p)
	d1, d2, d3 := e.createDocument(t, c, p, ev1), e.createDocument(t, c, p, ev2), e.createDocument(t, c, p, "")
	line := e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": p})
	e.trash(t, "/api/v1/cases/"+c+"/people/"+p)

	require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", "/api/v1/cases/"+c+"/people/"+p+"/permanent", nil).Code)

	assert.Equal(t, []string{p}, e.permanentlyDeleted(t, c, "person"))
	assert.ElementsMatch(t, []string{ev1, ev2}, e.permanentlyDeleted(t, c, "life_event"))
	assert.ElementsMatch(t, []string{d1, d2, d3}, e.permanentlyDeleted(t, c, "document"))
	assert.ElementsMatch(t, []string{line}, e.permanentlyDeleted(t, c, "claim_line"))
	// Parent links are logged against their child.
	assert.ElementsMatch(t, []string{p, child}, e.permanentlyDeleted(t, c, "person_relationship"))
}

// Scenario 15
func TestLog_PermanentDeleteLifeEvent_LogsItsDocuments(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	ev := e.createLifeEvent(t, c, p)
	d1, d2 := e.createDocument(t, c, p, ev), e.createDocument(t, c, p, ev)
	other := e.createDocument(t, c, p, "") // not under the event, so it stays
	e.trash(t, "/api/v1/cases/"+c+"/life-events/"+ev)

	require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", "/api/v1/cases/"+c+"/life-events/"+ev+"/permanent", nil).Code)

	assert.Equal(t, []string{ev}, e.permanentlyDeleted(t, c, "life_event"))
	assert.ElementsMatch(t, []string{d1, d2}, e.permanentlyDeleted(t, c, "document"))
	assert.NotContains(t, e.permanentlyDeleted(t, c, "document"), other)
}
