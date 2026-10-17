// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cases_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/cases"
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
	return e.c.Do(method, path, body)
}

func (e *testEnv) createCase(t *testing.T, title string) *cases.Case {
	t.Helper()
	var c cases.Case
	e.c.MustCreateInto(t, "/api/v1/cases", map[string]any{"title": title}, &c)
	return &c
}

func (e *testEnv) deleteCase(t *testing.T, caseID string) {
	t.Helper()
	e.c.MustDelete(t, "/api/v1/cases/"+caseID)
}

// ---------------------------------------------------------------------------
// POST /api/v1/cases
// ---------------------------------------------------------------------------

func TestCreateCase_Valid(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("POST", "/api/v1/cases", map[string]string{"title": "Test Case"})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got cases.Case
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, "Test Case", got.Title)
	assert.Equal(t, "active", got.Status)
}

func TestCreateCase_EmptyTitle(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("POST", "/api/v1/cases", map[string]string{"title": ""})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateCase_MissingTitle(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("POST", "/api/v1/cases", map[string]any{})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/cases/:caseId
// ---------------------------------------------------------------------------

func TestUpdateCase_Title(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "Original Title")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID, map[string]string{"title": "Updated Title"})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got cases.Case
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "Updated Title", got.Title)
	assert.Equal(t, "active", got.Status)
}

func TestUpdateCase_Status(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "My Case")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID, map[string]string{"status": "archived"})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got cases.Case
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "My Case", got.Title)
	assert.Equal(t, "archived", got.Status)
}

func TestUpdateCase_BothFields(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "Original")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID, map[string]string{
		"title": "Renamed", "status": "complete",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got cases.Case
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "Renamed", got.Title)
	assert.Equal(t, "complete", got.Status)
}

func TestUpdateCase_EmptyBody(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "Unchanged")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID, map[string]any{})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got cases.Case
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "Unchanged", got.Title)
	assert.Equal(t, "active", got.Status)
}

func TestUpdateCase_EmptyTitle(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "My Case")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID, map[string]string{"title": ""})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateCase_InvalidStatus(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "My Case")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID, map[string]string{"status": "bogus"})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateCase_NotFound(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("PATCH", "/api/v1/cases/00000000-0000-0000-0000-000000000000", map[string]string{"title": "X"})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateCase_OnSoftDeletedCase(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "About to be deleted")
	env.deleteCase(t, c.ID)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID, map[string]string{"title": "New"})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/cases/:caseId
// ---------------------------------------------------------------------------

func TestDeleteCase_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "To Be Deleted")

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID, nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	getRec := env.do("GET", "/api/v1/cases/"+c.ID, nil)
	assert.Equal(t, http.StatusNotFound, getRec.Code)
}

func TestDeleteCase_NotFound(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("DELETE", "/api/v1/cases/00000000-0000-0000-0000-000000000000", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteCase_AlreadyDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "Double Delete")
	env.deleteCase(t, c.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// Soft-delete visibility on existing reads
// ---------------------------------------------------------------------------

func TestListCases_ExcludesDeleted(t *testing.T) {
	env := newTestEnv(t)
	env.createCase(t, "Visible")
	hidden := env.createCase(t, "Hidden")
	env.deleteCase(t, hidden.ID)

	rec := env.do("GET", "/api/v1/cases", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Items []cases.Case `json:"items"`
		Total int          `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Items, 1)
	assert.Equal(t, "Visible", got.Items[0].Title)
}

func TestGetCase_OnSoftDeletedCase(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t, "Deleted Case")
	env.deleteCase(t, c.ID)

	rec := env.do("GET", "/api/v1/cases/"+c.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
