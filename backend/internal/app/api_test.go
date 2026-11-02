// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the API conventions settled in Milestone 3: record
// versions, one update per entity, complete lists, field-level errors, and
// deletion warnings (api-conventions.md).

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out), "body: %s", rec.Body.String())
	return out
}

// get fetches one record from its case's list, failing if it is not there.
func (e *env) get(t *testing.T, caseID, list, id string) map[string]any {
	t.Helper()
	item := findByID(e.list(t, "/api/v1/cases/"+caseID+"/"+list), id)
	require.NotNil(t, item, "%s %s not listed", list, id)
	return item
}

func version(t *testing.T, record map[string]any) int {
	t.Helper()
	v, ok := record["version"].(float64)
	require.True(t, ok, "record has no numeric version: %v", record)
	return int(v)
}

type fieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func fieldErrors(t *testing.T, rec *httptest.ResponseRecorder) (string, []fieldError) {
	t.Helper()
	var body struct {
		Error struct {
			Code   string       `json:"code"`
			Fields []fieldError `json:"fields"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "body: %s", rec.Body.String())
	return body.Error.Code, body.Error.Fields
}

func errorFor(errs []fieldError, field string) *fieldError {
	for i := range errs {
		if errs[i].Field == field {
			return &errs[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// A. Record versions
// ---------------------------------------------------------------------------

// Scenario 1
func TestVersions_StartAtOneAndRiseWithEachChange(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	ev := e.createLifeEvent(t, c, p)
	doc := e.createDocument(t, c, p, ev)
	line := e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": p})

	records := []struct{ name, path, list, id string }{
		{"person", "/api/v1/cases/" + c + "/people/" + p, "people", p},
		{"life event", "/api/v1/cases/" + c + "/life-events/" + ev, "life-events", ev},
		{"document", "/api/v1/cases/" + c + "/documents/" + doc, "documents", doc},
		{"claim line", "/api/v1/cases/" + c + "/claim-lines/" + line, "claim-lines", line},
	}
	changes := map[string]map[string]any{
		"person":     {"notes": "x"},
		"life event": {"event_place": "Palermo"},
		"document":   {"notes": "x"},
		"claim line": {"notes": "x"},
	}
	for _, r := range records {
		t.Run(r.name, func(t *testing.T) {
			assert.Equal(t, 1, version(t, e.get(t, c, r.list, r.id)))
			body := changes[r.name]
			body["version"] = 1
			rec := e.c.Do("PATCH", r.path, body)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Equal(t, 2, version(t, decode(t, rec)), "the response carries the new version")
			assert.Equal(t, 2, version(t, e.get(t, c, r.list, r.id)))
		})
	}

	t.Run("case", func(t *testing.T) {
		rec := e.c.Do("GET", "/api/v1/cases/"+c, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, 1, version(t, decode(t, rec)))
		rec = e.c.Do("PATCH", "/api/v1/cases/"+c, map[string]any{"title": "Renamed", "version": 1})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, 2, version(t, decode(t, rec)))
	})
}

// Scenario 2
func TestVersions_StaleUpdateRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	path := "/api/v1/cases/" + c + "/people/" + p
	require.Equal(t, http.StatusOK, e.c.Do("PATCH", path, map[string]any{"notes": "first", "version": 1}).Code)

	rec := e.c.Do("PATCH", path, map[string]any{"first_name": "Joseph", "version": 1}) // based on version 1, now 2

	assert.Equal(t, http.StatusConflict, rec.Code)
	code, _ := errorBody(t, rec)
	assert.Equal(t, "version_conflict", code)
	assert.Equal(t, "Giuseppe", e.get(t, c, "people", p)["first_name"], "the stale update was saved")
}

// Scenario 3
func TestVersions_MissingVersionRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	path := "/api/v1/cases/" + c + "/people/" + p

	t.Run("update", func(t *testing.T) {
		rec := e.c.Do("PATCH", path, map[string]any{"first_name": "Joseph"})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		code, errs := fieldErrors(t, rec)
		assert.Equal(t, "invalid_input", code)
		fe := errorFor(errs, "version")
		require.NotNil(t, fe, "no error for the missing version")
		assert.Equal(t, "required", fe.Code)
	})
	t.Run("delete", func(t *testing.T) {
		rec := e.c.Do("DELETE", path, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.NotNil(t, findByID(e.list(t, "/api/v1/cases/"+c+"/people"), p), "the person was trashed without a version")
	})
}

// Scenario 4
func TestVersions_StaleTrashRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	path := "/api/v1/cases/" + c + "/people/" + p
	require.Equal(t, http.StatusOK, e.c.Do("PATCH", path, map[string]any{"notes": "edited", "version": 1}).Code)

	rec := e.c.Do("DELETE", path+"?version=1", nil)

	assert.Equal(t, http.StatusConflict, rec.Code)
	code, _ := errorBody(t, rec)
	assert.Equal(t, "version_conflict", code)
	assert.NotNil(t, findByID(e.list(t, "/api/v1/cases/"+c+"/people"), p))
}

// ---------------------------------------------------------------------------
// B. One update per entity
// ---------------------------------------------------------------------------

// Scenario 5
func TestDocumentUpdate_ChangesEverythingAtOnce(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	a, b := e.createPerson(t, c, "Antonio"), e.createPerson(t, c, "Carlo")
	evB := e.createLifeEvent(t, c, b)
	doc := e.createDocument(t, c, a, "")

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc, map[string]any{
		"version":                 1,
		"title":                   "Marriage certificate",
		"person_id":               b,
		"life_event_id":           evB,
		"official_copy_status_id": e.statusID(t, "official_copy", "Requested"),
		"translation_status_id":   e.statusID(t, "translation", "Not Required"),
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	d := e.get(t, c, "documents", doc)
	assert.Equal(t, "Marriage certificate", d["title"])
	assert.Equal(t, b, d["person_id"])
	assert.Equal(t, evB, d["life_event_id"])
	assert.Equal(t, "Requested", d["official_copy_status"].(map[string]any)["label"])
	assert.Equal(t, "Not Required", d["translation_status"].(map[string]any)["label"])
}

// Scenario 6
func TestDocumentUpdate_AllOrNothing(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	doc := e.createDocument(t, c, p, "")

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc, map[string]any{
		"version":               1,
		"title":                 "Changed",
		"apostille_status_id":   e.statusID(t, "apostille", "Sent"),
		"translation_status_id": e.statusID(t, "official_copy", "Requested"), // wrong phase
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	d := e.get(t, c, "documents", doc)
	assert.Equal(t, "Birth certificate", d["title"], "the title was saved although the request was refused")
	assert.Equal(t, "Not Started", d["apostille_status"].(map[string]any)["label"])
	assert.Equal(t, 1, version(t, d))
}

// Scenario 7
func TestPersonUpdate_SetsDetailsAndFamilyLinksTogether(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Antonio")
	father, mother, child := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Maria"), e.createPerson(t, c, "Carlo")
	path := "/api/v1/cases/" + c + "/people/" + p

	rec := e.c.Do("PATCH", path, map[string]any{
		"version": 1, "first_name": "Anthony",
		"parent_ids": []string{father, mother}, "child_ids": []string{child},
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "Anthony", e.get(t, c, "people", p)["first_name"])
	assert.ElementsMatch(t, []string{father, mother}, e.parentIDs(t, c, p))
	assert.Equal(t, []string{p}, e.parentIDs(t, c, child))

	t.Run("lists replace what was there", func(t *testing.T) {
		rec := e.c.Do("PATCH", path, map[string]any{"version": 2, "parent_ids": []string{father}, "child_ids": []string{}})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, []string{father}, e.parentIDs(t, c, p))
		assert.Empty(t, e.parentIDs(t, c, child))
	})

	t.Run("a refused link saves nothing", func(t *testing.T) {
		// child as a parent of their own child... the loop is refused, and the name change with it.
		rec := e.c.Do("PATCH", path, map[string]any{"version": 3, "first_name": "Tony", "child_ids": []string{father}})
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "Anthony", e.get(t, c, "people", p)["first_name"])
		assert.Empty(t, e.parentIDs(t, c, father))
	})
}

// Scenario 8
func TestPersonCreate_WithFamilyLinks(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	father, child := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Carlo")

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people", map[string]any{
		"first_name": "Antonio", "last_name": "Rossi",
		"parent_ids": []string{father}, "child_ids": []string{child},
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	p := decode(t, rec)["id"].(string)

	assert.Equal(t, []string{father}, e.parentIDs(t, c, p))
	assert.Equal(t, []string{p}, e.parentIDs(t, c, child))
}

// Scenario 9
func TestLifeEventUpdate_MovesToAnotherPerson(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	a, b := e.createPerson(t, c, "Antonio"), e.createPerson(t, c, "Carlo")
	ev := e.createLifeEvent(t, c, a)
	doc := e.createDocument(t, c, a, ev)

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/life-events/"+ev,
		map[string]any{"version": 1, "person_id": b, "event_place": "Boston"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	le := e.get(t, c, "life-events", ev)
	assert.Equal(t, b, le["person_id"])
	assert.Equal(t, "Boston", le["event_place"])
	assert.Equal(t, b, e.get(t, c, "documents", doc)["person_id"])
}

// ---------------------------------------------------------------------------
// C. Lists
// ---------------------------------------------------------------------------

// Scenario 10. The count is chosen above any page size the code might fall
// back on, so the test can tell "all of them" from "the first page".
func TestLists_ReturnEveryRecord(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	const n = 87
	for i := 0; i < n; i++ {
		e.c.MustCreate(t, "/api/v1/cases/"+c+"/people",
			map[string]any{"first_name": fmt.Sprintf("Person%02d", i), "last_name": "Rossi"})
	}

	rec := e.c.Do("GET", "/api/v1/cases/"+c+"/people", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	assert.Len(t, body["items"], n)
	assert.EqualValues(t, n, body["total"])
}

// ---------------------------------------------------------------------------
// D. Errors
// ---------------------------------------------------------------------------

// Scenario 11
func TestErrors_EveryInvalidFieldReported(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/documents", map[string]any{
		"document_type": "not_a_type", "title": "",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	code, errs := fieldErrors(t, rec)
	assert.Equal(t, "invalid_input", code)
	for _, field := range []string{"person_id", "document_type", "title"} {
		fe := errorFor(errs, field)
		if assert.NotNil(t, fe, "no error for %s", field) {
			assert.NotEmpty(t, fe.Code, field)
			assert.NotEmpty(t, fe.Message, field)
		}
	}
}

// Scenario 12
func TestErrors_MalformedIDIsInvalidInput(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{
		"/api/v1/cases/not-an-id",
		"/api/v1/cases/not-an-id/people",
	} {
		rec := e.c.Do("GET", path, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code, path)
		code, _ := errorBody(t, rec)
		assert.Equal(t, "invalid_input", code, path)
	}
}

// Scenario 13. Only YYYY-MM-DD is accepted, so a date is never reinterpreted:
// the database would otherwise read "03/04/1890" month-first as 4 March.
func TestErrors_InvalidDateIsFieldError(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)

	for _, date := range []string{
		"15 March 1890", // written out
		"03/04/1890",    // ambiguous: 3 April or 4 March
		"1890-02-30",    // the right form, but no such day
	} {
		t.Run(date, func(t *testing.T) {
			rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people",
				map[string]any{"first_name": "Giuseppe", "last_name": "Rossi", "birth_date": date})

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			_, errs := fieldErrors(t, rec)
			fe := errorFor(errs, "birth_date")
			if assert.NotNil(t, fe) {
				assert.Equal(t, "invalid_date", fe.Code)
			}
		})
	}
	assert.Empty(t, e.list(t, "/api/v1/cases/"+c+"/people"), "a person was saved with an invalid date")
}

// Scenario 14
func TestErrors_UnknownFieldRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people",
		map[string]any{"first_name": "Giuseppe", "last_name": "Rossi", "brith_date": "1890-03-15"})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	_, errs := fieldErrors(t, rec)
	fe := errorFor(errs, "brith_date")
	require.NotNil(t, fe, "the misspelt field is not named")
	assert.Equal(t, "unknown_field", fe.Code)
	assert.Empty(t, e.list(t, "/api/v1/cases/"+c+"/people"))
}

// Scenario 15
func TestErrors_NamesTrimmedAndBlankIsRequired(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people", map[string]any{"first_name": "  Giuseppe  ", "last_name": "Rossi"})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, "Giuseppe", decode(t, rec)["first_name"])

	rec = e.c.Do("POST", "/api/v1/cases/"+c+"/people", map[string]any{"first_name": "   ", "last_name": "Rossi"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	_, errs := fieldErrors(t, rec)
	fe := errorFor(errs, "first_name")
	require.NotNil(t, fe)
	assert.Equal(t, "required", fe.Code)
}

// ---------------------------------------------------------------------------
// E. Deletion warnings
// ---------------------------------------------------------------------------

// Scenario 16
func TestDeletionImpact(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	ev1, ev2 := e.createLifeEvent(t, c, p), e.createLifeEvent(t, c, p)
	e.createDocument(t, c, p, ev1)
	e.createDocument(t, c, p, ev1)
	e.createDocument(t, c, p, "")
	e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": p})

	t.Run("person", func(t *testing.T) {
		rec := e.c.Do("GET", "/api/v1/cases/"+c+"/people/"+p+"/deletion-impact", nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, map[string]any{"life_events": 2.0, "documents": 3.0, "claim_lines": 1.0}, decode(t, rec))
	})
	t.Run("life event", func(t *testing.T) {
		rec := e.c.Do("GET", "/api/v1/cases/"+c+"/life-events/"+ev1+"/deletion-impact", nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, map[string]any{"documents": 2.0}, decode(t, rec))

		rec = e.c.Do("GET", "/api/v1/cases/"+c+"/life-events/"+ev2+"/deletion-impact", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, map[string]any{"documents": 0.0}, decode(t, rec))
	})
}

func (e *env) impact(t *testing.T, path string) map[string]any {
	t.Helper()
	rec := e.c.Do("GET", path+"/deletion-impact", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return decode(t, rec)
}

func (e *env) permanentlyDelete(t *testing.T, path string) {
	t.Helper()
	rec := e.c.Do("DELETE", path+"/permanent", nil)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
}

// Scenario 17: for a person in the trash, the counts are everything a permanent
// delete removes, including records trashed separately beforehand.
func TestDeletionImpact_TrashedPersonCountsEverythingPermanentlyDeleted(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	base := "/api/v1/cases/" + c
	p := e.createPerson(t, c, "Giuseppe")
	ev1, ev2 := e.createLifeEvent(t, c, p), e.createLifeEvent(t, c, p)
	e.createDocument(t, c, p, ev1)
	d2 := e.createDocument(t, c, p, ev1)
	e.createDocument(t, c, p, ev2)
	e.createDocument(t, c, p, "")
	cl1 := e.c.MustCreate(t, base+"/claim-lines", map[string]any{"root_person_id": p})
	e.trash(t, base+"/claim-lines/"+cl1)
	e.c.MustCreate(t, base+"/claim-lines", map[string]any{"root_person_id": p})
	e.trash(t, base+"/documents/"+d2)
	e.trash(t, base+"/life-events/"+ev2)
	e.trash(t, base+"/people/"+p)

	got := e.impact(t, base+"/people/"+p)
	assert.Equal(t, map[string]any{"life_events": 2.0, "documents": 4.0, "claim_lines": 2.0}, got)

	e.permanentlyDelete(t, base+"/people/"+p)
	assert.Len(t, e.permanentlyDeleted(t, c, "life_event"), int(got["life_events"].(float64)), "life events removed")
	assert.Len(t, e.permanentlyDeleted(t, c, "document"), int(got["documents"].(float64)), "documents removed")
	assert.Len(t, e.permanentlyDeleted(t, c, "claim_line"), int(got["claim_lines"].(float64)), "claim lines removed")
}

// Scenario 18: for a life event in the trash, the count is every document a
// permanent delete removes, including ones trashed separately.
func TestDeletionImpact_TrashedLifeEventCountsEverythingPermanentlyDeleted(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	base := "/api/v1/cases/" + c
	p := e.createPerson(t, c, "Giuseppe")
	ev := e.createLifeEvent(t, c, p)
	e.createDocument(t, c, p, ev)
	d2 := e.createDocument(t, c, p, ev)
	e.trash(t, base+"/documents/"+d2)
	e.trash(t, base+"/life-events/"+ev)

	got := e.impact(t, base+"/life-events/"+ev)
	assert.Equal(t, map[string]any{"documents": 2.0}, got)

	e.permanentlyDelete(t, base+"/life-events/"+ev)
	assert.Len(t, e.permanentlyDeleted(t, c, "document"), 2, "documents removed")
}

// Scenario 19: for an active record, records already in the trash are not
// counted, since they do not move.
func TestDeletionImpact_ActiveRecordLeavesOutTrashedChildren(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	base := "/api/v1/cases/" + c
	p := e.createPerson(t, c, "Giuseppe")
	ev1, ev2 := e.createLifeEvent(t, c, p), e.createLifeEvent(t, c, p)
	e.createDocument(t, c, p, ev1)
	d2 := e.createDocument(t, c, p, ev1)
	cl1 := e.c.MustCreate(t, base+"/claim-lines", map[string]any{"root_person_id": p})
	e.trash(t, base+"/claim-lines/"+cl1)
	e.c.MustCreate(t, base+"/claim-lines", map[string]any{"root_person_id": p})
	e.trash(t, base+"/documents/"+d2)
	e.trash(t, base+"/life-events/"+ev2)

	assert.Equal(t, map[string]any{"life_events": 1.0, "documents": 1.0, "claim_lines": 1.0}, e.impact(t, base+"/people/"+p))
	assert.Equal(t, map[string]any{"documents": 1.0}, e.impact(t, base+"/life-events/"+ev1))
}

// Scenario 20: a record beneath something in the trash, but not in the trash
// itself, cannot be deleted on its own, so there is nothing to report.
func TestDeletionImpact_BeneathTrashedParentNotFound(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	base := "/api/v1/cases/" + c
	p := e.createPerson(t, c, "Giuseppe")
	ev := e.createLifeEvent(t, c, p)
	e.trash(t, base+"/people/"+p)

	rec := e.c.Do("GET", base+"/life-events/"+ev+"/deletion-impact", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

// Scenario 21: a family link belongs to both people it joins, so changing it
// raises both versions. A stale save of the other person is then refused
// instead of silently removing the link.
func TestFamilyLink_RaisesBothVersions(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	x, carlo := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Carlo")
	xPath := "/api/v1/cases/" + c + "/people/" + x
	carloPath := "/api/v1/cases/" + c + "/people/" + carlo

	rec := e.c.Do("PATCH", carloPath, map[string]any{"version": 1, "parent_ids": []string{x}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, 2, version(t, e.get(t, c, "people", carlo)))
	assert.Equal(t, 2, version(t, e.get(t, c, "people", x)), "the parent's version did not rise")

	// A save of X based on the version from before the link, with the child list as it was then.
	rec = e.c.Do("PATCH", xPath, map[string]any{"version": 1, "child_ids": []string{}})
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "version_conflict", decode(t, rec)["error"].(map[string]any)["code"])
	assert.Equal(t, []string{x}, e.parentIDs(t, c, carlo), "the link was removed by a stale save")

	t.Run("removing a link raises both too", func(t *testing.T) {
		rec := e.c.Do("PATCH", carloPath, map[string]any{"version": 2, "parent_ids": []string{}})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, 3, version(t, e.get(t, c, "people", carlo)))
		assert.Equal(t, 3, version(t, e.get(t, c, "people", x)))
	})
}
