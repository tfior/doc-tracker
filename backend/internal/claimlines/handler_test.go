package claimlines_test

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
	"github.com/tfior/doc-tracker/internal/claimlines"
	"github.com/tfior/doc-tracker/internal/people"
	"github.com/tfior/doc-tracker/internal/testhelpers"
)

type testEnv struct {
	svc       *claimlines.Service
	casesSvc  *cases.Service
	peopleSvc *people.Service
	mux       *http.ServeMux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	testhelpers.TruncateCases(t, db)

	casesSvc := cases.NewService(cases.NewStore(db))
	peopleSvc := people.NewService(people.NewStore(db))
	clSvc := claimlines.NewService(claimlines.NewStore(db))

	mux := http.NewServeMux()
	claimlines.NewHandler(clSvc).RegisterRoutes(mux)

	return &testEnv{svc: clSvc, casesSvc: casesSvc, peopleSvc: peopleSvc, mux: mux}
}

func (e *testEnv) createCase(t *testing.T) *cases.Case {
	t.Helper()
	c, err := e.casesSvc.CreateCase(context.Background(), "Test Case")
	require.NoError(t, err)
	return c
}

func (e *testEnv) createPerson(t *testing.T, caseID string) people.Person {
	t.Helper()
	p, err := e.peopleSvc.CreatePerson(context.Background(), caseID, "Giuseppe", "Rossi", people.UpdatePersonInput{})
	require.NoError(t, err)
	return *p
}

// createClaimLine creates a claim line via the HTTP endpoint.
// Fails the test immediately if the endpoint returns a non-201 status.
func (e *testEnv) createClaimLine(t *testing.T, caseID, rootPersonID string) claimlines.ClaimLine {
	t.Helper()
	rec := e.do("POST", "/api/v1/cases/"+caseID+"/claim-lines", map[string]any{
		"root_person_id": rootPersonID,
	})
	require.Equal(t, http.StatusCreated, rec.Code, "setup: createClaimLine failed")
	var cl claimlines.ClaimLine
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&cl))
	return cl
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
// POST /api/v1/cases/:caseId/claim-lines
// ---------------------------------------------------------------------------

func TestCreateClaimLine_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/claim-lines", map[string]any{
		"root_person_id": p.ID,
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got claimlines.ClaimLine
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, c.ID, got.CaseID)
	assert.Equal(t, p.ID, got.RootPersonID)
	assert.Equal(t, "not_yet_researched", got.Status)
	assert.Nil(t, got.Notes)
}

func TestCreateClaimLine_WithStatus(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/claim-lines", map[string]any{
		"root_person_id": p.ID,
		"status":         "eligible",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got claimlines.ClaimLine
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "eligible", got.Status)
}

func TestCreateClaimLine_WithNotes(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/claim-lines", map[string]any{
		"root_person_id": p.ID,
		"notes":          "Primary line through paternal grandfather",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got claimlines.ClaimLine
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.Notes)
	assert.Equal(t, "Primary line through paternal grandfather", *got.Notes)
}

func TestCreateClaimLine_MissingRootPersonID(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/claim-lines", map[string]any{})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateClaimLine_InvalidStatus(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/claim-lines", map[string]any{
		"root_person_id": p.ID,
		"status":         "bogus",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateClaimLine_PersonNotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/claim-lines", map[string]any{
		"root_person_id": "00000000-0000-0000-0000-000000000000",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateClaimLine_PersonNotInCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	personInB := env.createPerson(t, caseB.ID)

	rec := env.do("POST", "/api/v1/cases/"+caseA.ID+"/claim-lines", map[string]any{
		"root_person_id": personInB.ID,
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/cases/:caseId/claim-lines/:lineId
// ---------------------------------------------------------------------------

func TestUpdateClaimLine_Status(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	cl := env.createClaimLine(t, c.ID, p.ID)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, map[string]any{
		"status": "researching",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got claimlines.ClaimLine
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "researching", got.Status)
}

func TestUpdateClaimLine_Notes(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	cl := env.createClaimLine(t, c.ID, p.ID)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, map[string]any{
		"notes": "Secondary maternal line",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got claimlines.ClaimLine
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.Notes)
	assert.Equal(t, "Secondary maternal line", *got.Notes)
}

func TestUpdateClaimLine_ClearNotes(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	createRec := env.do("POST", "/api/v1/cases/"+c.ID+"/claim-lines", map[string]any{
		"root_person_id": p.ID,
		"notes":          "Some note",
	})
	require.Equal(t, http.StatusCreated, createRec.Code)
	var cl claimlines.ClaimLine
	require.NoError(t, json.NewDecoder(createRec.Body).Decode(&cl))

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, map[string]any{
		"notes": nil,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got claimlines.ClaimLine
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Nil(t, got.Notes)
}

func TestUpdateClaimLine_InvalidStatus(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	cl := env.createClaimLine(t, c.ID, p.ID)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, map[string]any{
		"status": "bogus",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateClaimLine_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/claim-lines/00000000-0000-0000-0000-000000000000", map[string]any{
		"status": "researching",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestUpdateClaimLine_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID)
	cl := env.createClaimLine(t, caseA.ID, p.ID)

	rec := env.do("PATCH", "/api/v1/cases/"+caseB.ID+"/claim-lines/"+cl.ID, map[string]any{
		"status": "researching",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateClaimLine_SoftDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	cl := env.createClaimLine(t, c.ID, p.ID)

	delRec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, nil)
	require.Equal(t, http.StatusNoContent, delRec.Code)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, map[string]any{
		"status": "researching",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/cases/:caseId/claim-lines/:lineId
// ---------------------------------------------------------------------------

func TestDeleteClaimLine_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	cl := env.createClaimLine(t, c.ID, p.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/claim-lines", nil)
	var got struct{ Total int `json:"total"` }
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&got))
	assert.Equal(t, 0, got.Total)
}

func TestDeleteClaimLine_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/claim-lines/00000000-0000-0000-0000-000000000000", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestDeleteClaimLine_AlreadyDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	cl := env.createClaimLine(t, c.ID, p.ID)

	env.do("DELETE", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, nil)
	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/claim-lines/"+cl.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteClaimLine_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID)
	cl := env.createClaimLine(t, caseA.ID, p.ID)

	rec := env.do("DELETE", "/api/v1/cases/"+caseB.ID+"/claim-lines/"+cl.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// Soft-delete visibility on list
// ---------------------------------------------------------------------------

func TestListClaimLines_ExcludesDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	env.createClaimLine(t, c.ID, p.ID)
	hidden := env.createClaimLine(t, c.ID, p.ID)

	delRec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/claim-lines/"+hidden.ID, nil)
	require.Equal(t, http.StatusNoContent, delRec.Code)

	rec := env.do("GET", "/api/v1/cases/"+c.ID+"/claim-lines", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Items []claimlines.ClaimLine `json:"items"`
		Total int                    `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Items, 1)
}
