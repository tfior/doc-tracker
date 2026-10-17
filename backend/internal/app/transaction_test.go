// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package app_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/testhelpers"
)

// These tests cover the rule that a write and its activity log entry are saved
// together or not at all (architecture.md, Transactions; domain model, Activity Log).

type env struct {
	db *sql.DB
	c  *testhelpers.AppClient
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	testhelpers.TruncateCases(t, db)
	testhelpers.TruncateUsers(t, db)
	return &env{db: db, c: testhelpers.NewAppClient(t, db)}
}

func (e *env) createCase(t *testing.T) string {
	t.Helper()
	return e.c.MustCreate(t, "/api/v1/cases", map[string]any{"title": "Test Case"})
}

func (e *env) createPerson(t *testing.T, caseID, firstName string) string {
	t.Helper()
	return e.c.MustCreate(t, "/api/v1/cases/"+caseID+"/people",
		map[string]any{"first_name": firstName, "last_name": "Rossi"})
}

// entriesFor returns the actions logged for one entity, oldest first, and
// fails the test if any of them was recorded against a different user.
func (e *env) entriesFor(t *testing.T, caseID, entityType, entityID string) []string {
	t.Helper()
	actions := []string{}
	for _, entry := range testhelpers.ActivityLog(t, e.db, caseID) {
		if entry.EntityType != entityType || entry.EntityID != entityID {
			continue
		}
		assert.Equal(t, e.c.UserID, entry.UserID, "%s %s logged against the wrong user", entityType, entry.Action)
		actions = append(actions, entry.Action)
	}
	return actions
}

func decodeItems(t *testing.T, rec *http.Response) []map[string]any {
	t.Helper()
	var out struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	return out.Items
}

func (e *env) list(t *testing.T, path string) []map[string]any {
	t.Helper()
	rec := e.c.Do("GET", path, nil)
	require.Equal(t, http.StatusOK, rec.Code, "GET %s", path)
	return decodeItems(t, rec.Result())
}

func findByID(items []map[string]any, id string) map[string]any {
	for _, it := range items {
		if it["id"] == id {
			return it
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Scenario 1: each create, update and delete writes exactly one log entry,
// with the right action and the logged-in user.
// ---------------------------------------------------------------------------

func TestActivityLog_OneEntryPerWrite(t *testing.T) {
	type tc struct {
		entityType string
		// run performs the writes and returns the case ID and the logged entity's ID.
		run  func(t *testing.T, e *env) (caseID, entityID string)
		want []string
	}
	cases := []tc{
		{
			entityType: "case",
			run: func(t *testing.T, e *env) (string, string) {
				id := e.createCase(t)
				require.Equal(t, http.StatusOK, e.c.Do("PATCH", "/api/v1/cases/"+id, map[string]any{"title": "Renamed"}).Code)
				require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", "/api/v1/cases/"+id, nil).Code)
				return id, id
			},
			want: []string{"created", "updated", "deleted"},
		},
		{
			entityType: "person",
			run: func(t *testing.T, e *env) (string, string) {
				caseID := e.createCase(t)
				id := e.createPerson(t, caseID, "Giuseppe")
				p := "/api/v1/cases/" + caseID + "/people/" + id
				require.Equal(t, http.StatusOK, e.c.Do("PATCH", p, map[string]any{"first_name": "Joseph"}).Code)
				require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", p, nil).Code)
				return caseID, id
			},
			want: []string{"created", "updated", "deleted"},
		},
		{
			entityType: "person_relationship",
			run: func(t *testing.T, e *env) (string, string) {
				caseID := e.createCase(t)
				child := e.createPerson(t, caseID, "Antonio")
				parent := e.createPerson(t, caseID, "Giuseppe")
				p := "/api/v1/cases/" + caseID + "/people/" + child + "/relationships"
				require.Equal(t, http.StatusCreated, e.c.Do("POST", p, map[string]any{"parent_id": parent}).Code)
				require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", p+"/"+parent, nil).Code)
				return caseID, child
			},
			want: []string{"created", "deleted"},
		},
		{
			entityType: "life_event",
			run: func(t *testing.T, e *env) (string, string) {
				caseID := e.createCase(t)
				person := e.createPerson(t, caseID, "Giuseppe")
				id := e.c.MustCreate(t, "/api/v1/cases/"+caseID+"/life-events",
					map[string]any{"person_id": person, "event_type": "birth"})
				p := "/api/v1/cases/" + caseID + "/life-events/" + id
				require.Equal(t, http.StatusOK, e.c.Do("PATCH", p, map[string]any{"event_place": "Palermo"}).Code)
				require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", p, nil).Code)
				return caseID, id
			},
			want: []string{"created", "updated", "deleted"},
		},
		{
			entityType: "document",
			run: func(t *testing.T, e *env) (string, string) {
				caseID := e.createCase(t)
				person := e.createPerson(t, caseID, "Giuseppe")
				id := e.c.MustCreate(t, "/api/v1/cases/"+caseID+"/documents",
					map[string]any{"person_id": person, "document_type": "birth_certificate", "title": "Birth certificate"})
				p := "/api/v1/cases/" + caseID + "/documents/" + id
				require.Equal(t, http.StatusOK, e.c.Do("PATCH", p, map[string]any{"title": "Italian birth certificate"}).Code)
				require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", p, nil).Code)
				return caseID, id
			},
			want: []string{"created", "updated", "deleted"},
		},
		{
			entityType: "claim_line",
			run: func(t *testing.T, e *env) (string, string) {
				caseID := e.createCase(t)
				root := e.createPerson(t, caseID, "Giuseppe")
				id := e.c.MustCreate(t, "/api/v1/cases/"+caseID+"/claim-lines",
					map[string]any{"root_person_id": root})
				p := "/api/v1/cases/" + caseID + "/claim-lines/" + id
				require.Equal(t, http.StatusOK, e.c.Do("PATCH", p, map[string]any{"status": "researching"}).Code)
				require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", p, nil).Code)
				return caseID, id
			},
			want: []string{"created", "updated", "deleted"},
		},
	}
	for _, c := range cases {
		t.Run(c.entityType, func(t *testing.T) {
			e := newEnv(t)
			caseID, entityID := c.run(t, e)
			assert.Equal(t, c.want, e.entriesFor(t, caseID, c.entityType, entityID))
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: a write that fails leaves no log entry behind.
// ---------------------------------------------------------------------------

func TestActivityLog_NoEntryWhenWriteFails(t *testing.T) {
	e := newEnv(t)
	caseID := e.createCase(t)
	before := testhelpers.ActivityLog(t, e.db, caseID)

	missing := "00000000-0000-0000-0000-000000000001"
	rec := e.c.Do("PATCH", "/api/v1/cases/"+caseID+"/people/"+missing, map[string]any{"first_name": "Nobody"})
	require.Equal(t, http.StatusNotFound, rec.Code)

	assert.Equal(t, before, testhelpers.ActivityLog(t, e.db, caseID))
}

// ---------------------------------------------------------------------------
// Scenarios 3 to 6: if the log entry cannot be written, the change is not saved.
// ---------------------------------------------------------------------------

func TestCreatePerson_LogFails_NothingSaved(t *testing.T) {
	e := newEnv(t)
	caseID := e.createCase(t)
	testhelpers.FailActivityLogWrites(t, e.db)

	rec := e.c.Do("POST", "/api/v1/cases/"+caseID+"/people", map[string]any{"first_name": "Giuseppe", "last_name": "Rossi"})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Empty(t, e.list(t, "/api/v1/cases/"+caseID+"/people"), "person was saved although its log entry failed")
}

func TestUpdatePerson_LogFails_NothingSaved(t *testing.T) {
	e := newEnv(t)
	caseID := e.createCase(t)
	personID := e.createPerson(t, caseID, "Giuseppe")
	testhelpers.FailActivityLogWrites(t, e.db)

	rec := e.c.Do("PATCH", "/api/v1/cases/"+caseID+"/people/"+personID, map[string]any{"first_name": "Joseph"})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	p := findByID(e.list(t, "/api/v1/cases/"+caseID+"/people"), personID)
	require.NotNil(t, p)
	assert.Equal(t, "Giuseppe", p["first_name"], "update was saved although its log entry failed")
}

func TestDeletePerson_LogFails_NothingSaved(t *testing.T) {
	e := newEnv(t)
	caseID := e.createCase(t)
	personID := e.createPerson(t, caseID, "Giuseppe")
	testhelpers.FailActivityLogWrites(t, e.db)

	rec := e.c.Do("DELETE", "/api/v1/cases/"+caseID+"/people/"+personID, nil)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotNil(t, findByID(e.list(t, "/api/v1/cases/"+caseID+"/people"), personID),
		"person was moved to the trash although its log entry failed")
}

// Reassigning a document changes two fields together. Neither may be saved if
// the log entry fails. (The reassignment endpoint is replaced by the document
// update in the API consistency branch; this test moves with it.)
func TestReassignDocument_LogFails_NothingSaved(t *testing.T) {
	e := newEnv(t)
	caseID := e.createCase(t)
	personA := e.createPerson(t, caseID, "Antonio")
	personB := e.createPerson(t, caseID, "Carlo")
	eventB := e.c.MustCreate(t, "/api/v1/cases/"+caseID+"/life-events",
		map[string]any{"person_id": personB, "event_type": "birth"})
	docID := e.c.MustCreate(t, "/api/v1/cases/"+caseID+"/documents",
		map[string]any{"person_id": personA, "document_type": "birth_certificate", "title": "Birth certificate"})
	testhelpers.FailActivityLogWrites(t, e.db)

	rec := e.c.Do("PATCH", "/api/v1/cases/"+caseID+"/documents/"+docID+"/parent",
		map[string]any{"person_id": personB, "life_event_id": eventB})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	d := findByID(e.list(t, "/api/v1/cases/"+caseID+"/documents"), docID)
	require.NotNil(t, d)
	assert.Equal(t, personA, d["person_id"], "person changed although the log entry failed")
	assert.Nil(t, d["life_event_id"], "life event changed although the log entry failed")
}
