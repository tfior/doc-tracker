package lifeevents_test

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
	"github.com/tfior/doc-tracker/internal/lifeevents"
	"github.com/tfior/doc-tracker/internal/people"
	"github.com/tfior/doc-tracker/internal/testhelpers"
)

type testEnv struct {
	svc      *lifeevents.Service
	casesSvc *cases.Service
	peopleSvc *people.Service
	mux      *http.ServeMux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	testhelpers.TruncateCases(t, db)

	casesSvc := cases.NewService(cases.NewStore(db))
	peopleSvc := people.NewService(people.NewStore(db))
	leSvc := lifeevents.NewService(lifeevents.NewStore(db))

	mux := http.NewServeMux()
	lifeevents.NewHandler(leSvc, nil).RegisterRoutes(mux)

	return &testEnv{svc: leSvc, casesSvc: casesSvc, peopleSvc: peopleSvc, mux: mux}
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

// createLifeEvent creates a life event via the HTTP endpoint.
// Fails the test immediately if the endpoint returns a non-201 status.
func (e *testEnv) createLifeEvent(t *testing.T, caseID, personID, eventType string) lifeevents.LifeEvent {
	t.Helper()
	rec := e.do("POST", "/api/v1/cases/"+caseID+"/life-events", map[string]any{
		"person_id":  personID,
		"event_type": eventType,
	})
	require.Equal(t, http.StatusCreated, rec.Code, "setup: createLifeEvent failed")
	var le lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&le))
	return le
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
// POST /api/v1/cases/:caseId/life-events
// ---------------------------------------------------------------------------

func TestCreateLifeEvent_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/life-events", map[string]any{
		"person_id":  p.ID,
		"event_type": "birth",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, c.ID, got.CaseID)
	assert.Equal(t, p.ID, got.PersonID)
	assert.Equal(t, "birth", got.EventType)
	assert.Nil(t, got.EventDate)
	assert.Nil(t, got.EventPlace)
	assert.Nil(t, got.SpouseName)
}

func TestCreateLifeEvent_WithOptionalFields(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/life-events", map[string]any{
		"person_id":          p.ID,
		"event_type":         "marriage",
		"event_date":         "1920-06-15",
		"event_place":        "Naples, Italy",
		"spouse_name":        "Maria Conti",
		"spouse_birth_date":  "1900-03-01",
		"spouse_birth_place": "Rome, Italy",
		"notes":              "First marriage",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	var got lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.EventDate)
	assert.Equal(t, "1920-06-15", *got.EventDate)
	require.NotNil(t, got.EventPlace)
	assert.Equal(t, "Naples, Italy", *got.EventPlace)
	require.NotNil(t, got.SpouseName)
	assert.Equal(t, "Maria Conti", *got.SpouseName)
	require.NotNil(t, got.SpouseBirthDate)
	assert.Equal(t, "1900-03-01", *got.SpouseBirthDate)
	require.NotNil(t, got.SpouseBirthPlace)
	assert.Equal(t, "Rome, Italy", *got.SpouseBirthPlace)
	require.NotNil(t, got.Notes)
	assert.Equal(t, "First marriage", *got.Notes)
}

func TestCreateLifeEvent_MissingPersonID(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/life-events", map[string]any{
		"event_type": "birth",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateLifeEvent_MissingEventType(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/life-events", map[string]any{
		"person_id": p.ID,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateLifeEvent_InvalidEventType(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/life-events", map[string]any{
		"person_id":  p.ID,
		"event_type": "graduation",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateLifeEvent_UnknownPerson(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/life-events", map[string]any{
		"person_id":  "00000000-0000-0000-0000-000000000000",
		"event_type": "birth",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateLifeEvent_PersonNotInCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	personInB := env.createPerson(t, caseB.ID)

	rec := env.do("POST", "/api/v1/cases/"+caseA.ID+"/life-events", map[string]any{
		"person_id":  personInB.ID,
		"event_type": "birth",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/cases/:caseId/life-events/:eventId
// ---------------------------------------------------------------------------

func TestUpdateLifeEvent_EventType(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, map[string]any{
		"event_type": "death",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "death", got.EventType)
}

func TestUpdateLifeEvent_OptionalFields(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, map[string]any{
		"event_date":  "1895-04-10",
		"event_place": "Palermo, Italy",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.NotNil(t, got.EventDate)
	assert.Equal(t, "1895-04-10", *got.EventDate)
	require.NotNil(t, got.EventPlace)
	assert.Equal(t, "Palermo, Italy", *got.EventPlace)
}

func TestUpdateLifeEvent_ClearOptionalField(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	// Create with event_date set, then explicitly clear it.
	rec := env.do("POST", "/api/v1/cases/"+c.ID+"/life-events", map[string]any{
		"person_id":  p.ID,
		"event_type": "birth",
		"event_date": "1895-04-10",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var le lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&le))

	patchRec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, map[string]any{
		"event_date": nil,
	})

	assert.Equal(t, http.StatusOK, patchRec.Code)
	var got lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(patchRec.Body).Decode(&got))
	assert.Nil(t, got.EventDate)
}

func TestUpdateLifeEvent_InvalidEventType(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, map[string]any{
		"event_type": "graduation",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateLifeEvent_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/00000000-0000-0000-0000-000000000000", map[string]any{
		"event_type": "death",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "not_found", body.Error.Code)
}

func TestUpdateLifeEvent_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID)
	le := env.createLifeEvent(t, caseA.ID, p.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+caseB.ID+"/life-events/"+le.ID, map[string]any{
		"event_type": "death",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateLifeEvent_SoftDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID, "birth")

	delRec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, nil)
	require.Equal(t, http.StatusNoContent, delRec.Code)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, map[string]any{
		"event_type": "death",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/cases/:caseId/life-events/:eventId
// ---------------------------------------------------------------------------

func TestDeleteLifeEvent_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID, "birth")

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	listRec := env.do("GET", "/api/v1/cases/"+c.ID+"/life-events", nil)
	var got struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&got))
	assert.Equal(t, 0, got.Total)
}

func TestDeleteLifeEvent_NotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)

	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/life-events/00000000-0000-0000-0000-000000000000", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "not_found", body.Error.Code)
}

func TestDeleteLifeEvent_AlreadyDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID, "birth")

	env.do("DELETE", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, nil)
	rec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteLifeEvent_WrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p := env.createPerson(t, caseA.ID)
	le := env.createLifeEvent(t, caseA.ID, p.ID, "birth")

	rec := env.do("DELETE", "/api/v1/cases/"+caseB.ID+"/life-events/"+le.ID, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/cases/:caseId/life-events/:eventId/person (reassign)
// ---------------------------------------------------------------------------

func TestReassignLifeEvent_Valid(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p1 := env.createPerson(t, c.ID)
	p2 := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p1.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID+"/person", map[string]any{
		"person_id": p2.ID,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, p2.ID, got.PersonID)
}

func TestReassignLifeEvent_SamePerson(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID+"/person", map[string]any{
		"person_id": p.ID,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	var got lifeevents.LifeEvent
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, p.ID, got.PersonID)
}

func TestReassignLifeEvent_PersonNotInCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p1 := env.createPerson(t, caseA.ID)
	p2 := env.createPerson(t, caseB.ID)
	le := env.createLifeEvent(t, caseA.ID, p1.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+caseA.ID+"/life-events/"+le.ID+"/person", map[string]any{
		"person_id": p2.ID,
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestReassignLifeEvent_PersonNotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	le := env.createLifeEvent(t, c.ID, p.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/"+le.ID+"/person", map[string]any{
		"person_id": "00000000-0000-0000-0000-000000000000",
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestReassignLifeEvent_EventNotFound(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	rec := env.do("PATCH", "/api/v1/cases/"+c.ID+"/life-events/00000000-0000-0000-0000-000000000000/person", map[string]any{
		"person_id": p.ID,
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "not_found", body.Error.Code)
}

func TestReassignLifeEvent_EventWrongCase(t *testing.T) {
	env := newTestEnv(t)
	caseA := env.createCase(t)
	caseB := env.createCase(t)
	p1 := env.createPerson(t, caseA.ID)
	p2 := env.createPerson(t, caseB.ID)
	le := env.createLifeEvent(t, caseA.ID, p1.ID, "birth")

	rec := env.do("PATCH", "/api/v1/cases/"+caseB.ID+"/life-events/"+le.ID+"/person", map[string]any{
		"person_id": p2.ID,
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// Soft-delete visibility on list
// ---------------------------------------------------------------------------

func TestListLifeEvents_ExcludesDeleted(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)
	env.createLifeEvent(t, c.ID, p.ID, "birth")
	hidden := env.createLifeEvent(t, c.ID, p.ID, "death")

	delRec := env.do("DELETE", "/api/v1/cases/"+c.ID+"/life-events/"+hidden.ID, nil)
	require.Equal(t, http.StatusNoContent, delRec.Code)

	rec := env.do("GET", "/api/v1/cases/"+c.ID+"/life-events", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Items []lifeevents.LifeEvent `json:"items"`
		Total int                    `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Items, 1)
	assert.Equal(t, "birth", got.Items[0].EventType)
}
