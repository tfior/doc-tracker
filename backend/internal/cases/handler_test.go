package cases_test

import (
	"bytes"
	"context"
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
	svc *cases.Service
	mux *http.ServeMux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	testhelpers.TruncateCases(t, db)

	store := cases.NewStore(db)
	svc := cases.NewService(store)

	mux := http.NewServeMux()
	cases.NewHandler(svc, nil).RegisterRoutes(mux)

	return &testEnv{svc: svc, mux: mux}
}

func (e *testEnv) createCase(t *testing.T, title string) *cases.Case {
	t.Helper()
	c, err := e.svc.CreateCase(context.Background(), title)
	require.NoError(t, err)
	return c
}

func (e *testEnv) do(method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
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
	require.NoError(t, env.svc.DeleteCase(context.Background(), c.ID))

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
	require.NoError(t, env.svc.DeleteCase(context.Background(), c.ID))

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
	require.NoError(t, env.svc.DeleteCase(context.Background(), hidden.ID))

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
	require.NoError(t, env.svc.DeleteCase(context.Background(), c.ID))

	rec := env.do("GET", "/api/v1/cases/"+c.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
