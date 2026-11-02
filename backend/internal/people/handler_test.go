// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package people_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/cases"
	"github.com/tfior/doc-tracker/internal/people"
	"github.com/tfior/doc-tracker/internal/testhelpers"
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
	return e.c.DoV(method, path, body)
}

func (e *testEnv) createCase(t *testing.T) *cases.Case {
	t.Helper()
	var c cases.Case
	e.c.MustCreateInto(t, "/api/v1/cases", map[string]any{"title": "Test Case"}, &c)
	return &c
}

// createPerson creates a person via the HTTP endpoint and returns the decoded Person.
// Fails the test immediately if the endpoint returns a non-201 status.
func (e *testEnv) createPerson(t *testing.T, caseID, firstName, lastName string) people.Person {
	t.Helper()
	rec := e.do("POST", "/api/v1/cases/"+caseID+"/people", map[string]any{
		"first_name": firstName,
		"last_name":  lastName,
	})
	require.Equal(t, http.StatusCreated, rec.Code, "setup: createPerson failed")
	var p people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))
	return p
}

// ---------------------------------------------------------------------------
// POST /api/v1/cases/:caseId/people
// ---------------------------------------------------------------------------

func TestCreatePerson_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/people", map[string]any{
		"first_name": "Giuseppe",
		"last_name":  "Rossi",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, c.ID, got.CaseID)
	assert.Equal(t, "Giuseppe", got.FirstName)
	assert.Equal(t, "Rossi", got.LastName)
	assert.Nil(t, got.BirthDate)
	assert.Nil(t, got.BirthPlace)
	assert.Nil(t, got.DeathDate)
	assert.Nil(t, got.Notes)
}

func TestCreatePerson_WithOptionalFields(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/people", map[string]any{
		"first_name":  "Antonio",
		"last_name":   "Rossi",
		"birth_date":  "1890-03-15",
		"birth_place": "Rome, Italy",
		"death_date":  "1955-11-01",
		"notes":       "LIRA candidate",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.BirthDate)
	assert.Equal(t, "1890-03-15", *got.BirthDate)
	require.NotNil(t, got.BirthPlace)
	assert.Equal(t, "Rome, Italy", *got.BirthPlace)
	require.NotNil(t, got.DeathDate)
	assert.Equal(t, "1955-11-01", *got.DeathDate)
	require.NotNil(t, got.Notes)
	assert.Equal(t, "LIRA candidate", *got.Notes)
}

func TestCreatePerson_MissingFirstName(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/people", map[string]any{
		"last_name": "Rossi",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePerson_MissingLastName(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/people", map[string]any{
		"first_name": "Giuseppe",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePerson_UnknownCase(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("POST", "/api/v1/cases/00000000-0000-0000-0000-000000000000/people", map[string]any{
		"first_name": "Giuseppe",
		"last_name":  "Rossi",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/cases/:caseId/people/:personId
// ---------------------------------------------------------------------------

func TestUpdatePerson_Name(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/people/"+p.ID, map[string]any{
		"first_name": "Gino",
		"last_name":  "Bianchi",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "Gino", got.FirstName)
	assert.Equal(t, "Bianchi", got.LastName)
}

func TestUpdatePerson_OptionalFields(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/people/"+p.ID, map[string]any{
		"birth_date":  "1880-06-01",
		"birth_place": "Naples, Italy",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.BirthDate)
	assert.Equal(t, "1880-06-01", *got.BirthDate)
	require.NotNil(t, got.BirthPlace)
	assert.Equal(t, "Naples, Italy", *got.BirthPlace)
}

func TestUpdatePerson_ClearOptionalField(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	// Create with a birth_date set, then explicitly clear it.
	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/people", map[string]any{
		"first_name": "Giuseppe",
		"last_name":  "Rossi",
		"birth_date": "1880-06-01",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var p people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))

	patchRec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/people/"+p.ID, map[string]any{
		"birth_date": nil,
	})

	assert.Equal(t, http.StatusOK, patchRec.Code)
	var got people.Person
	require.NoError(t, json.NewDecoder(patchRec.Body).Decode(&got))
	assert.Nil(t, got.BirthDate)
}

func TestUpdatePerson_EmptyFirstName(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/people/"+p.ID, map[string]any{
		"first_name": "",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdatePerson_EmptyLastName(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/people/"+p.ID, map[string]any{
		"last_name": "",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdatePerson_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/people/00000000-0000-0000-0000-000000000000", map[string]any{
		"first_name": "X",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdatePerson_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID, "Giuseppe", "Rossi")

	// Attempt to update person from case A using case B's URL.
	rec := env.do("PATCH", "/api/v1/cases/"+caseB.ID+"/people/"+p.ID, map[string]any{
		"first_name": "X",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdatePerson_SoftDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	delRec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/"+p.ID, nil)
	require.Equal(t, http.StatusNoContent, delRec.Code)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/people/"+p.ID, map[string]any{
		"first_name": "X",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/cases/:caseId/people/:personId
// ---------------------------------------------------------------------------

func TestDeletePerson_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/"+p.ID, nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/people", nil)
	var got struct {
		Items []people.Person `json:"items"`
		Total int             `json:"total"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&got))
	assert.Equal(t, 0, got.Total)
}

func TestDeletePerson_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/00000000-0000-0000-0000-000000000000", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeletePerson_AlreadyDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/"+p.ID, nil)
	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/"+p.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeletePerson_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID, "Giuseppe", "Rossi")

	rec := env.do("DELETE", "/api/v1/cases/"+caseB.ID+"/people/"+p.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// Parent links, set through PATCH /api/v1/cases/:caseId/people/:personId
// ---------------------------------------------------------------------------

// setParents sets a person's complete list of parents through the person update.
func (e *testEnv) setParents(caseID, personID string, parentIDs ...string) *httptest.ResponseRecorder {
	if parentIDs == nil {
		parentIDs = []string{}
	}
	return e.do("PATCH", "/api/v1/cases/"+caseID+"/people/"+personID, map[string]any{"parent_ids": parentIDs})
}

func TestSetParents_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	child := env.createPerson(t, c.ID, "Antonio", "Rossi")
	parent := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	rec := env.setParents(c.ID, child.ID, parent.ID)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, []string{parent.ID}, got.ParentIDs)
}

func TestSetParents_SelfReference(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	rec := env.setParents(c.ID, p.ID, p.ID)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetParents_ParentInDifferentCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	child := env.createPerson(t, caseA.ID, "Antonio", "Rossi")
	parent := env.createPerson(t, caseB.ID, "Giuseppe", "Rossi")

	rec := env.setParents(caseA.ID, child.ID, parent.ID)

	// The API is case-scoped; a parent from a different case doesn't exist in
	// this case's context, so the response is 404 rather than 400.
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSetParents_ExceedsMaxParents(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	child := env.createPerson(t, c.ID, "Antonio", "Rossi")
	parent1 := env.createPerson(t, c.ID, "Giuseppe", "Rossi")
	parent2 := env.createPerson(t, c.ID, "Maria", "Ferrari")
	parent3 := env.createPerson(t, c.ID, "Carlo", "Rossi")

	require.Equal(t, http.StatusOK, env.setParents(c.ID, child.ID, parent1.ID, parent2.ID).Code)

	rec := env.setParents(c.ID, child.ID, parent1.ID, parent2.ID, parent3.ID)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestSetParents_PersonNotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	parent := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	rec := env.setParents(c.ID, "00000000-0000-0000-0000-000000000000", parent.ID)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSetParents_ParentNotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	child := env.createPerson(t, c.ID, "Antonio", "Rossi")

	rec := env.setParents(c.ID, child.ID, "00000000-0000-0000-0000-000000000000")

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSetParents_RemoveParent(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	child := env.createPerson(t, c.ID, "Antonio", "Rossi")
	parent := env.createPerson(t, c.ID, "Giuseppe", "Rossi")
	require.Equal(t, http.StatusOK, env.setParents(c.ID, child.ID, parent.ID).Code)

	rec := env.setParents(c.ID, child.ID)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Empty(t, got.ParentIDs)
}

// ---------------------------------------------------------------------------
// Soft-delete visibility on list
// ---------------------------------------------------------------------------

func TestListPeople_ExcludesDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	env.createPerson(t, c.ID, "Giuseppe", "Rossi")
	hidden := env.createPerson(t, c.ID, "Antonio", "Rossi")

	delRec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/people/"+hidden.ID, nil)
	require.Equal(t, http.StatusNoContent, delRec.Code)

	rec := env.do("GET", "/api/v1/cases/"+c.ID+"/people", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Items []people.Person `json:"items"`
		Total int             `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Items, 1)
	assert.Equal(t, "Giuseppe", got.Items[0].FirstName)
}

// ---------------------------------------------------------------------------
// parent_ids field
// ---------------------------------------------------------------------------

func TestParentIDs_NewPersonHasEmptySlice(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")

	assert.NotNil(t, p.ParentIDs)
	assert.Empty(t, p.ParentIDs)
}

func TestParentIDs_ReflectedInList(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	parent := env.createPerson(t, c.ID, "Giuseppe", "Rossi")
	child := env.createPerson(t, c.ID, "Antonio", "Rossi")

	require.Equal(t, http.StatusOK, env.setParents(c.ID, child.ID, parent.ID).Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/people", nil)
	var list struct {
		Items []people.Person `json:"items"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&list))

	for _, p := range list.Items {
		if p.ID == child.ID {
			assert.Contains(t, p.ParentIDs, parent.ID)
		}
		if p.ID == parent.ID {
			assert.Empty(t, p.ParentIDs)
		}
	}
}

func TestParentIDs_ClearedAfterRemoval(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	parent := env.createPerson(t, c.ID, "Giuseppe", "Rossi")
	child := env.createPerson(t, c.ID, "Antonio", "Rossi")

	require.Equal(t, http.StatusOK, env.setParents(c.ID, child.ID, parent.ID).Code)
	require.Equal(t, http.StatusOK, env.setParents(c.ID, child.ID).Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/people", nil)
	var list struct {
		Items []people.Person `json:"items"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&list))

	for _, p := range list.Items {
		if p.ID == child.ID {
			assert.Empty(t, p.ParentIDs)
		}
	}
}

func TestParentIDs_PresentOnCreateAndUpdateResponse(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	p := env.createPerson(t, c.ID, "Giuseppe", "Rossi")
	assert.NotNil(t, p.ParentIDs)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/people/"+p.ID, map[string]any{
		"first_name": "Giuseppe",
	})
	assert.Equal(t, http.StatusOK, rec.Code)
	var updated people.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&updated))
	assert.NotNil(t, updated.ParentIDs)
}
