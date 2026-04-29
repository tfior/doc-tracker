package documents_test

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
	"github.com/tfior/doc-tracker/internal/documents"
	"github.com/tfior/doc-tracker/internal/lifeevents"
	"github.com/tfior/doc-tracker/internal/people"
	"github.com/tfior/doc-tracker/internal/testhelpers"
)

type testEnv struct {
	svc       *documents.Service
	casesSvc  *cases.Service
	peopleSvc *people.Service
	leSvc     *lifeevents.Service
	mux       *http.ServeMux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	testhelpers.TruncateCases(t, db)

	casesSvc := cases.NewService(cases.NewStore(db))
	peopleSvc := people.NewService(people.NewStore(db))
	leSvc := lifeevents.NewService(lifeevents.NewStore(db))
	docSvc := documents.NewService(documents.NewStore(db))

	mux := http.NewServeMux()
	documents.NewHandler(docSvc, nil).RegisterRoutes(mux)

	return &testEnv{svc: docSvc, casesSvc: casesSvc, peopleSvc: peopleSvc, leSvc: leSvc, mux: mux}
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

func (e *testEnv) createLifeEvent(t *testing.T, caseID, personID string) lifeevents.LifeEvent {
	t.Helper()
	le, err := e.leSvc.CreateLifeEvent(context.Background(), caseID, personID, "birth", lifeevents.UpdateLifeEventInput{})
	require.NoError(t, err)
	return *le
}

// createDocument creates a document via the HTTP endpoint.
// Fails the test immediately if the endpoint returns a non-201 status.
func (e *testEnv) createDocument(t *testing.T, caseID, personID, docType, title string) documents.Document {
	t.Helper()
	rec := e.do("POST", "/api/v1/cases/"+caseID+"/documents", map[string]any{
		"person_id":     personID,
		"document_type": docType,
		"title":         title,
	})
	require.Equal(t, http.StatusCreated, rec.Code, "setup: createDocument failed")
	var d documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&d))
	return d
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
// POST /api/v1/cases/:caseId/documents
// ---------------------------------------------------------------------------

func TestCreateDocument_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"person_id":     p.ID,
		"document_type": "birth_certificate",
		"title":         "Giuseppe Rossi Birth Certificate",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, c.ID, got.CaseID)
	assert.Equal(t, p.ID, got.PersonID)
	assert.Equal(t, "birth_certificate", got.DocumentType)
	assert.Equal(t, "Giuseppe Rossi Birth Certificate", got.Title)
	assert.False(t, got.IsVerified)
	assert.Nil(t, got.VerifiedAt)
	assert.Nil(t, got.LifeEventID)
	// Status must default to the system "pending" status.
	assert.Equal(t, "pending", *got.StatusKey)
	assert.Equal(t, "not_started", got.ProgressBucket)
}

func TestCreateDocument_WithOptionalFields(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"person_id":            p.ID,
		"document_type":        "birth_certificate",
		"title":                "Giuseppe Birth Cert",
		"life_event_id":        le.ID,
		"issuing_authority":    "Comune di Napoli",
		"issue_date":           "1895-01-01",
		"recorded_given_name":  "Giuseppe",
		"recorded_surname":     "Rossi",
		"recorded_birth_date":  "1880-06-01",
		"recorded_birth_place": "Napoli",
		"notes":                "Original Italian document",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.LifeEventID)
	assert.Equal(t, le.ID, *got.LifeEventID)
	require.NotNil(t, got.IssuingAuthority)
	assert.Equal(t, "Comune di Napoli", *got.IssuingAuthority)
	require.NotNil(t, got.RecordedGivenName)
	assert.Equal(t, "Giuseppe", *got.RecordedGivenName)
	require.NotNil(t, got.RecordedBirthPlace)
	assert.Equal(t, "Napoli", *got.RecordedBirthPlace)
}

func TestCreateDocument_MissingPersonID(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"document_type": "birth_certificate",
		"title":         "Some Doc",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateDocument_MissingDocumentType(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"person_id": p.ID,
		"title":     "Some Doc",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateDocument_MissingTitle(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"person_id":     p.ID,
		"document_type": "birth_certificate",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateDocument_InvalidDocumentType(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"person_id":     p.ID,
		"document_type": "passport",
		"title":         "Some Doc",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateDocument_PersonNotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"person_id":     "00000000-0000-0000-0000-000000000000",
		"document_type": "birth_certificate",
		"title":         "Some Doc",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateDocument_PersonNotInCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	personInB := env.createPerson(t, caseB.ID)

	rec := env.do("POST", "/api/v1/cases/"+caseA.ID+"/documents", map[string]any{
		"person_id":     personInB.ID,
		"document_type": "birth_certificate",
		"title":         "Some Doc",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateDocument_LifeEventNotInCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	personA := env.createPerson(t, caseA.ID)
	personB := env.createPerson(t, caseB.ID)
	leInB := env.createLifeEvent(t, caseB.ID, personB.ID)

	rec := env.do("POST", "/api/v1/cases/"+caseA.ID+"/documents", map[string]any{
		"person_id":     personA.ID,
		"document_type": "birth_certificate",
		"title":         "Some Doc",
		"life_event_id": leInB.ID,
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/cases/:caseId/documents/:docId
// ---------------------------------------------------------------------------

func TestUpdateDocument_Title(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Original Title")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, map[string]any{
		"title": "Updated Title",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "Updated Title", got.Title)
}

func TestUpdateDocument_RecordedFields(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Giuseppe Birth Cert")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, map[string]any{
		"recorded_given_name": "Giuseppe",
		"recorded_surname":    "Rossi",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.RecordedGivenName)
	assert.Equal(t, "Giuseppe", *got.RecordedGivenName)
	require.NotNil(t, got.RecordedSurname)
	assert.Equal(t, "Rossi", *got.RecordedSurname)
}

func TestUpdateDocument_ClearOptionalField(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	createRec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"person_id":           p.ID,
		"document_type":       "birth_certificate",
		"title":               "Some Doc",
		"recorded_given_name": "Giuseppe",
	})
	require.Equal(t, http.StatusCreated, createRec.Code)
	var d documents.Document
	require.NoError(t, json.NewDecoder(createRec.Body).Decode(&d))

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, map[string]any{
		"recorded_given_name": nil,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Nil(t, got.RecordedGivenName)
}

func TestUpdateDocument_IsVerified(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Some Doc")

	// Set verified.
	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, map[string]any{
		"is_verified": true,
	})
	assert.Equal(t, http.StatusOK, rec.Code)
	var verified documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&verified))
	assert.True(t, verified.IsVerified)
	assert.NotNil(t, verified.VerifiedAt)

	// Clear verified.
	rec2 := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, map[string]any{
		"is_verified": false,
	})
	assert.Equal(t, http.StatusOK, rec2.Code)
	var unverified documents.Document
	require.NoError(t, json.NewDecoder(rec2.Body).Decode(&unverified))
	assert.False(t, unverified.IsVerified)
	assert.Nil(t, unverified.VerifiedAt)
}

func TestUpdateDocument_InvalidDocumentType(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, map[string]any{
		"document_type": "passport",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateDocument_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/00000000-0000-0000-0000-000000000000", map[string]any{
		"title": "X",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestUpdateDocument_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID)
	d := env.createDocument(t, caseA.ID, p.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+caseB.ID+"/documents/"+d.ID, map[string]any{
		"title": "X",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateDocument_SoftDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Some Doc")

	delRec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, nil)
	require.Equal(t, http.StatusNoContent, delRec.Code)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, map[string]any{
		"title": "X",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/cases/:caseId/documents/:docId
// ---------------------------------------------------------------------------

func TestDeleteDocument_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Some Doc")

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/documents", nil)
	var got struct{ Total int `json:"total"` }
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&got))
	assert.Equal(t, 0, got.Total)
}

func TestDeleteDocument_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/documents/00000000-0000-0000-0000-000000000000", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestDeleteDocument_AlreadyDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Some Doc")

	env.do("DELETE", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, nil)
	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/documents/"+d.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteDocument_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID)
	d := env.createDocument(t, caseA.ID, p.ID, "birth_certificate", "Some Doc")

	rec := env.do("DELETE", "/api/v1/cases/"+caseB.ID+"/documents/"+d.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/cases/:caseId/documents/:docId/status
// ---------------------------------------------------------------------------

func TestTransitionStatus_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID+"/status", map[string]any{
		"status_key": "collected",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.StatusKey)
	assert.Equal(t, "collected", *got.StatusKey)
	assert.Equal(t, "in_progress", got.ProgressBucket)
}

func TestTransitionStatus_InvalidKey(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID+"/status", map[string]any{
		"status_key": "approved",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTransitionStatus_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/00000000-0000-0000-0000-000000000000/status", map[string]any{
		"status_key": "collected",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

func TestTransitionStatus_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID)
	d := env.createDocument(t, caseA.ID, p.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+caseB.ID+"/documents/"+d.ID+"/status", map[string]any{
		"status_key": "collected",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/cases/:caseId/documents/:docId/parent
// ---------------------------------------------------------------------------

func TestReassignDocument_ToPerson(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p1 := env.createPerson(t, c.ID)
	p2 := env.createPerson(t, c.ID)
	d := env.createDocument(t, c.ID, p1.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID+"/parent", map[string]any{
		"person_id": p2.ID,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, p2.ID, got.PersonID)
}

func TestReassignDocument_ToLifeEvent(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID)
	d := env.createDocument(t, c.ID, p.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID+"/parent", map[string]any{
		"person_id":     p.ID,
		"life_event_id": le.ID,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.LifeEventID)
	assert.Equal(t, le.ID, *got.LifeEventID)
}

func TestReassignDocument_ClearLifeEvent(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID)

	createRec := env.do("POST", "/api/v1/cases/"+c.ID+"/documents", map[string]any{
		"person_id":     p.ID,
		"document_type": "birth_certificate",
		"title":         "Some Doc",
		"life_event_id": le.ID,
	})
	require.Equal(t, http.StatusCreated, createRec.Code)
	var d documents.Document
	require.NoError(t, json.NewDecoder(createRec.Body).Decode(&d))

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/"+d.ID+"/parent", map[string]any{
		"person_id":     p.ID,
		"life_event_id": nil,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got documents.Document
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Nil(t, got.LifeEventID)
}

func TestReassignDocument_PersonNotInCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	pA := env.createPerson(t, caseA.ID)
	pB := env.createPerson(t, caseB.ID)
	d := env.createDocument(t, caseA.ID, pA.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+caseA.ID+"/documents/"+d.ID+"/parent", map[string]any{
		"person_id": pB.ID,
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestReassignDocument_LifeEventNotInCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	pA := env.createPerson(t, caseA.ID)
	pB := env.createPerson(t, caseB.ID)
	leInB := env.createLifeEvent(t, caseB.ID, pB.ID)
	d := env.createDocument(t, caseA.ID, pA.ID, "birth_certificate", "Some Doc")

	rec := env.do("PATCH", "/api/v1/cases/"+caseA.ID+"/documents/"+d.ID+"/parent", map[string]any{
		"person_id":     pA.ID,
		"life_event_id": leInB.ID,
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestReassignDocument_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/documents/00000000-0000-0000-0000-000000000000/parent", map[string]any{
		"person_id": p.ID,
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assertNotFoundBody(t, rec)
}

// ---------------------------------------------------------------------------
// Soft-delete visibility on list
// ---------------------------------------------------------------------------

func TestListDocuments_ExcludesDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	env.createDocument(t, c.ID, p.ID, "birth_certificate", "Visible Doc")
	hidden := env.createDocument(t, c.ID, p.ID, "death_certificate", "Hidden Doc")

	delRec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/documents/"+hidden.ID, nil)
	require.Equal(t, http.StatusNoContent, delRec.Code)

	rec := env.do("GET", "/api/v1/cases/"+c.ID+"/documents", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Items []documents.Document `json:"items"`
		Total int                  `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Items, 1)
	assert.Equal(t, "Visible Doc", got.Items[0].Title)
}
