// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package trash_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/cases"
	"github.com/tfior/doc-tracker/internal/claimlines"
	"github.com/tfior/doc-tracker/internal/documents"
	"github.com/tfior/doc-tracker/internal/lifeevents"
	"github.com/tfior/doc-tracker/internal/people"
	"github.com/tfior/doc-tracker/internal/testhelpers"
	"github.com/tfior/doc-tracker/internal/trash"
)

type testEnv struct {
	db *sql.DB
	c  *testhelpers.AppClient
}

// newTestEnv returns an environment that sends requests to the fully assembled
// application, behind the real authentication middleware, as a logged-in user.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	testhelpers.TruncateCases(t, db)
	testhelpers.TruncateUsers(t, db)
	return &testEnv{db: db, c: testhelpers.NewAppClient(t, db)}
}

func (e *testEnv) do(method, path string, body any) *httptest.ResponseRecorder {
	return e.c.Do(method, path, body)
}

// ---------------------------------------------------------------------------
// Setup helpers
// ---------------------------------------------------------------------------

func (e *testEnv) createCase(t *testing.T) *cases.Case {
	t.Helper()
	var c cases.Case
	e.c.MustCreateInto(t, "/api/v1/cases", map[string]any{"title": "Test Case"}, &c)
	return &c
}

func (e *testEnv) createPerson(t *testing.T, caseID string) *people.Person {
	t.Helper()
	var p people.Person
	e.c.MustCreateInto(t, "/api/v1/cases/"+caseID+"/people",
		map[string]any{"first_name": "Giuseppe", "last_name": "Rossi"}, &p)
	return &p
}

func (e *testEnv) createLifeEvent(t *testing.T, caseID, personID string) *lifeevents.LifeEvent {
	t.Helper()
	var le lifeevents.LifeEvent
	e.c.MustCreateInto(t, "/api/v1/cases/"+caseID+"/life-events",
		map[string]any{"person_id": personID, "event_type": "birth"}, &le)
	return &le
}

func (e *testEnv) createDocument(t *testing.T, caseID, personID string) *documents.Document {
	t.Helper()
	var d documents.Document
	e.c.MustCreateInto(t, "/api/v1/cases/"+caseID+"/documents",
		map[string]any{"person_id": personID, "document_type": "birth_certificate", "title": "Test Doc"}, &d)
	return &d
}

func (e *testEnv) createClaimLine(t *testing.T, caseID, personID string) *claimlines.ClaimLine {
	t.Helper()
	var cl claimlines.ClaimLine
	e.c.MustCreateInto(t, "/api/v1/cases/"+caseID+"/claim-lines",
		map[string]any{"root_person_id": personID}, &cl)
	return &cl
}

func (e *testEnv) deleteCase(t *testing.T, caseID string) {
	t.Helper()
	e.c.MustDelete(t, "/api/v1/cases/"+caseID)
}

func (e *testEnv) deletePerson(t *testing.T, caseID, personID string) {
	t.Helper()
	e.c.MustDelete(t, "/api/v1/cases/"+caseID+"/people/"+personID)
}

func (e *testEnv) deleteLifeEvent(t *testing.T, caseID, eventID string) {
	t.Helper()
	e.c.MustDelete(t, "/api/v1/cases/"+caseID+"/life-events/"+eventID)
}

func (e *testEnv) deleteDocument(t *testing.T, caseID, docID string) {
	t.Helper()
	e.c.MustDelete(t, "/api/v1/cases/"+caseID+"/documents/"+docID)
}

func (e *testEnv) deleteClaimLine(t *testing.T, caseID, lineID string) {
	t.Helper()
	e.c.MustDelete(t, "/api/v1/cases/"+caseID+"/claim-lines/"+lineID)
}

func assertNotFoundBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "not_found", body.Error.Code)
}

// ---------------------------------------------------------------------------
// GET /api/v1/cases/:caseId/trash
// ---------------------------------------------------------------------------

func TestGetCaseTrash_Empty(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("GET", "/api/v1/cases/"+c.ID+"/trash", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got trash.CaseTrash
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Empty(t, got.People)
	assert.Empty(t, got.LifeEvents)
	assert.Empty(t, got.Documents)
	assert.Empty(t, got.ClaimLines)
}

func TestGetCaseTrash_ShowsDeletedEntities(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID)

	// One active person, one trashed person; one trashed document.
	env.createPerson(t, c.ID) // active, should not appear
	// The document goes first: once its person is trashed it is frozen.
	env.deleteDocument(t, c.ID, d.ID)
	env.deletePerson(t, c.ID, p.ID)

	rec := env.do("GET", "/api/v1/cases/"+c.ID+"/trash", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got trash.CaseTrash
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Len(t, got.People, 1)
	assert.Equal(t, p.ID, got.People[0].ID)
	assert.Len(t, got.Documents, 1)
	assert.Equal(t, d.ID, got.Documents[0].ID)
	assert.Empty(t, got.LifeEvents)
	assert.Empty(t, got.ClaimLines)
}

func TestGetCaseTrash_ExcludesOtherCases(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	pA := env.createPerson(t, caseA.ID)
	pB := env.createPerson(t, caseB.ID)

	env.deletePerson(t, caseA.ID, pA.ID)
	env.deletePerson(t, caseB.ID, pB.ID)

	// Only case A's trash should appear.
	rec := env.do("GET", "/api/v1/cases/"+caseA.ID+"/trash", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got trash.CaseTrash
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Len(t, got.People, 1)
	assert.Equal(t, pA.ID, got.People[0].ID)
}

// ---------------------------------------------------------------------------
// Restore (case-scoped)
// ---------------------------------------------------------------------------

func TestRestorePerson_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	env.deletePerson(t, c.ID, p.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/people/"+p.ID+"/restore", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Person reappears in the people list.
	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/people", nil)
	var list struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&list))
	assert.Equal(t, 1, list.Total)

	// Person is gone from trash.
	trashRec := env.do("GET", "/api/v1/cases/"+c.ID+"/trash", nil)
	var tr trash.CaseTrash
	require.NoError(t, json.NewDecoder(trashRec.Body).Decode(&tr))
	assert.Empty(t, tr.People)
}

func TestRestorePerson_NotInTrash(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID) // active, not trashed

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/people/"+p.ID+"/restore", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestRestorePerson_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/people/00000000-0000-0000-0000-000000000000/restore", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestRestoreLifeEvent_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID)
	env.deleteLifeEvent(t, c.ID, le.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID+"/restore", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/life-events", nil)
	var list struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&list))
	assert.Equal(t, 1, list.Total)
}

func TestRestoreDocument_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID)
	env.deleteDocument(t, c.ID, d.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents/"+d.ID+"/restore", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/documents", nil)
	var list struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&list))
	assert.Equal(t, 1, list.Total)
}

func TestRestoreClaimLine_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	cl := env.createClaimLine(t, c.ID, p.ID)
	env.deleteClaimLine(t, c.ID, cl.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID+"/restore", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/claim-lines", nil)
	var list struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&list))
	assert.Equal(t, 1, list.Total)
}

// ---------------------------------------------------------------------------
// Permanent delete (case-scoped)
// ---------------------------------------------------------------------------

func TestPermanentDeletePerson_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	env.deletePerson(t, c.ID, p.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/"+p.ID+"/permanent", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Gone from trash.
	trashRec := env.do("GET", "/api/v1/cases/"+c.ID+"/trash", nil)
	var tr trash.CaseTrash
	require.NoError(t, json.NewDecoder(trashRec.Body).Decode(&tr))
	assert.Empty(t, tr.People)
}

func TestPermanentDeletePerson_CascadesToChildren(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	env.createLifeEvent(t, c.ID, p.ID)
	env.createDocument(t, c.ID, p.ID)
	env.deletePerson(t, c.ID, p.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/"+p.ID+"/permanent", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Life events and documents are also gone (FK cascade), not just in trash.
	leRec := env.do("GET", "/api/v1/cases/"+c.ID+"/life-events", nil)
	var leList struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(leRec.Body).Decode(&leList))
	assert.Equal(t, 0, leList.Total)

	docRec := env.do("GET", "/api/v1/cases/"+c.ID+"/documents", nil)
	var docList struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(docRec.Body).Decode(&docList))
	assert.Equal(t, 0, docList.Total)
}

func TestPermanentDeletePerson_NotInTrash(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID) // active, not trashed

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/"+p.ID+"/permanent", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestPermanentDeletePerson_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/00000000-0000-0000-0000-000000000000/permanent", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestPermanentDeleteLifeEvent_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID)
	env.deleteLifeEvent(t, c.ID, le.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID+"/permanent", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	trashRec := env.do("GET", "/api/v1/cases/"+c.ID+"/trash", nil)
	var tr trash.CaseTrash
	require.NoError(t, json.NewDecoder(trashRec.Body).Decode(&tr))
	assert.Empty(t, tr.LifeEvents)
}

func TestPermanentDeleteDocument_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID)
	env.deleteDocument(t, c.ID, d.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/documents/"+d.ID+"/permanent", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	trashRec := env.do("GET", "/api/v1/cases/"+c.ID+"/trash", nil)
	var tr trash.CaseTrash
	require.NoError(t, json.NewDecoder(trashRec.Body).Decode(&tr))
	assert.Empty(t, tr.Documents)
}

func TestPermanentDeleteClaimLine_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	cl := env.createClaimLine(t, c.ID, p.ID)
	env.deleteClaimLine(t, c.ID, cl.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID+"/permanent", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	trashRec := env.do("GET", "/api/v1/cases/"+c.ID+"/trash", nil)
	var tr trash.CaseTrash
	require.NoError(t, json.NewDecoder(trashRec.Body).Decode(&tr))
	assert.Empty(t, tr.ClaimLines)
}

// ---------------------------------------------------------------------------
// GET /api/v1/trash
// ---------------------------------------------------------------------------

func TestGetGlobalTrash_Empty(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("GET", "/api/v1/trash", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got trash.GlobalTrash
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Empty(t, got.Cases)
	assert.Empty(t, got.People)
	assert.Empty(t, got.LifeEvents)
	assert.Empty(t, got.Documents)
	assert.Empty(t, got.ClaimLines)
}

func TestGetGlobalTrash_ShowsTrashedCase(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	env.deleteCase(t, c.ID)

	rec := env.do("GET", "/api/v1/trash", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got trash.GlobalTrash
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Len(t, got.Cases, 1)
	assert.Equal(t, c.ID, got.Cases[0].ID)
}

func TestGetGlobalTrash_ShowsTrashedEntities(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	env.deletePerson(t, c.ID, p.ID)

	rec := env.do("GET", "/api/v1/trash", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got trash.GlobalTrash
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Empty(t, got.Cases)
	assert.Len(t, got.People, 1)
	assert.Equal(t, p.ID, got.People[0].ID)
}

func TestGetGlobalTrash_ExcludesEntitiesFromTrashedCases(t *testing.T) {
	env := newTestEnv(t)
	// Case A: active. Person in A is individually trashed — should appear.
	caseA := env.createCase(t)
	pA := env.createPerson(t, caseA.ID)
	env.deletePerson(t, caseA.ID, pA.ID)

	// Case B: trashed. Person in B is not individually trashed — should NOT appear.
	caseB := env.createCase(t)
	env.createPerson(t, caseB.ID)
	env.deleteCase(t, caseB.ID)

	rec := env.do("GET", "/api/v1/trash", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got trash.GlobalTrash
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Len(t, got.Cases, 1)
	assert.Equal(t, caseB.ID, got.Cases[0].ID)
	assert.Len(t, got.People, 1)
	assert.Equal(t, pA.ID, got.People[0].ID)
}

// ---------------------------------------------------------------------------
// Restore/permanent delete for cases
// ---------------------------------------------------------------------------

func TestRestoreCase_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	env.deleteCase(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/restore", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Case reappears in list.
	listRec := env.do("GET", "/api/v1/cases", nil)
	var list struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&list))
	assert.Equal(t, 1, list.Total)
}

func TestRestoreCase_NotInTrash(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t) // active, not trashed

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/restore", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestPermanentDeleteCase_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	env.createLifeEvent(t, c.ID, p.ID)
	env.deleteCase(t, c.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/permanent", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Case and all children are gone.
	listRec := env.do("GET", "/api/v1/cases", nil)
	var list struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&list))
	assert.Equal(t, 0, list.Total)

	globalRec := env.do("GET", "/api/v1/trash", nil)
	var gt trash.GlobalTrash
	require.NoError(t, json.NewDecoder(globalRec.Body).Decode(&gt))
	assert.Empty(t, gt.Cases)
}

func TestPermanentDeleteCase_NotInTrash(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t) // active, not trashed

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/permanent", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}
