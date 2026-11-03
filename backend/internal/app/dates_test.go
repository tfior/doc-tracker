// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package app_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/testhelpers"
)

// These tests cover partial and qualified dates, settled in Milestone 4: a
// document date transcribes what the document says, at whatever precision it
// gives; a life event date is a working belief with a qualifier
// (domain-model.md, Dates; api-conventions.md, Dates).

func (e *env) createDocumentWith(t *testing.T, caseID, personID string, fields map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"person_id": personID, "document_type": "birth_certificate", "title": "Birth certificate"}
	for k, v := range fields {
		body[k] = v
	}
	rec := e.c.Do("POST", "/api/v1/cases/"+caseID+"/documents", body)
	require.Equal(t, http.StatusCreated, rec.Code, "create document: %s", rec.Body.String())
	return decode(t, rec)
}

func (e *env) createEventWith(t *testing.T, caseID, personID string, fields map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"person_id": personID, "event_type": "birth"}
	for k, v := range fields {
		body[k] = v
	}
	rec := e.c.Do("POST", "/api/v1/cases/"+caseID+"/life-events", body)
	require.Equal(t, http.StatusCreated, rec.Code, "create life event: %s", rec.Body.String())
	return decode(t, rec)
}

func qualified(value, qualifier string) map[string]any {
	return map[string]any{"value": value, "qualifier": qualifier}
}

func between(value, end string) map[string]any {
	return map[string]any{"value": value, "qualifier": "between", "end": end}
}

var documentDateFields = []string{"issue_date", "registration_date", "recorded_birth_date"}

// ---------------------------------------------------------------------------
// A. Partial dates on documents
// ---------------------------------------------------------------------------

// Scenario 1
func TestPartialDate_EveryPrecisionAccepted(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	for _, date := range []string{"1890", "1890-03", "1890-03-15"} {
		t.Run(date, func(t *testing.T) {
			fields := map[string]any{}
			for _, f := range documentDateFields {
				fields[f] = date
			}
			doc := e.createDocumentWith(t, c, p, fields)
			for _, f := range documentDateFields {
				assert.Equal(t, date, doc[f], f)
			}
			listed := e.get(t, c, "documents", doc["id"].(string))
			for _, f := range documentDateFields {
				assert.Equal(t, date, listed[f], "%s as listed", f)
			}
		})
	}
}

// Scenario 2
func TestPartialDate_BadDatesRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	for _, date := range []string{
		"1890-3",           // month without its leading zero
		"15/03/1890",       // another form
		"90",               // two-digit year
		"1890-03-15T00:00", // a timestamp
		"0000",             // there is no year 0
		"1890-13",          // no such month
		"1890-02-30",       // no such day
		"1890-02-29",       // 1890 is not a leap year
	} {
		t.Run(date, func(t *testing.T) {
			rec := e.c.Do("POST", "/api/v1/cases/"+c+"/documents", map[string]any{
				"person_id": p, "document_type": "birth_certificate", "title": "Birth certificate",
				"issue_date": date, "registration_date": date, "recorded_birth_date": date,
			})
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			code, errs := fieldErrors(t, rec)
			assert.Equal(t, "invalid_input", code)
			for _, f := range documentDateFields {
				fe := errorFor(errs, f)
				if assert.NotNil(t, fe, "no error for %s", f) {
					assert.Equal(t, "invalid_date", fe.Code, f)
					assert.NotEmpty(t, fe.Message, f)
				}
			}
		})
	}

	doc := e.createDocumentWith(t, c, p, map[string]any{"issue_date": "1892-02-29"})
	assert.Equal(t, "1892-02-29", doc["issue_date"], "1892 is a leap year")
	assert.Len(t, e.list(t, "/api/v1/cases/"+c+"/documents"), 1, "a document was saved with a bad date")
}

// ---------------------------------------------------------------------------
// B. Qualified dates on life events
// ---------------------------------------------------------------------------

// Scenario 3
func TestQualifiedDate_EveryQualifierAccepted(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	dates := []map[string]any{
		qualified("1890-03-15", "exact"),
		qualified("1890-03", "exact"),
		qualified("1890", "about"),
		qualified("1915", "before"),
		qualified("1888-06", "after"),
		between("1890", "1892"),
		between("1890-03-15", "1890-04"),
	}
	for _, date := range dates {
		t.Run(date["qualifier"].(string)+" "+date["value"].(string), func(t *testing.T) {
			le := e.createEventWith(t, c, p, map[string]any{
				"event_type": "marriage", "event_date": date, "spouse_birth_date": date,
			})
			assert.Equal(t, date, le["event_date"])
			assert.Equal(t, date, le["spouse_birth_date"])
			listed := e.get(t, c, "life-events", le["id"].(string))
			assert.Equal(t, date, listed["event_date"], "as listed")
			assert.Equal(t, date, listed["spouse_birth_date"], "as listed")
		})
	}
}

// Scenario 4
func TestQualifiedDate_BadDatesRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	cases := []struct {
		name  string
		date  any
		field string // the part at fault, after "event_date"
		code  string
	}{
		{"no qualifier", map[string]any{"value": "1890"}, ".qualifier", "required"},
		{"unknown qualifier", qualified("1890", "circa"), ".qualifier", "invalid_value"},
		{"range without end", qualified("1890", "between"), ".end", "required"},
		{"end without range", map[string]any{"value": "1890", "qualifier": "about", "end": "1892"}, ".end", "invalid_value"},
		{"end equal to start", between("1890-03", "1890-03"), ".end", "invalid_value"},
		{"end inside start", between("1890", "1890-06"), ".end", "invalid_value"},
		{"end before start", between("1892", "1890"), ".end", "invalid_value"},
		{"bad value", qualified("1890-02-30", "exact"), ".value", "invalid_date"},
		{"bad end", between("1890", "1892-13"), ".end", "invalid_date"},
		{"no value", map[string]any{"qualifier": "exact"}, ".value", "required"},
		{"empty end", between("1890", ""), ".end", "required"},
		{"unknown key", map[string]any{"value": "1890", "qualifier": "exact", "precision": "year"}, ".precision", "unknown_field"},
		{"plain string", "1890-03-15", "", "invalid_value"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events", map[string]any{
				"person_id": p, "event_type": "marriage",
				"event_date": tc.date, "spouse_birth_date": tc.date,
			})
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			code, errs := fieldErrors(t, rec)
			assert.Equal(t, "invalid_input", code)
			for _, f := range []string{"event_date", "spouse_birth_date"} {
				fe := errorFor(errs, f+tc.field)
				if assert.NotNil(t, fe, "no error for %s%s in %v", f, tc.field, errs) {
					assert.Equal(t, tc.code, fe.Code)
					assert.NotEmpty(t, fe.Message)
				}
			}
		})
	}
	assert.Empty(t, e.list(t, "/api/v1/cases/"+c+"/life-events"), "a life event was saved with a bad date")
}

// Scenario 4, continued: every problem in one date is reported at once.
func TestQualifiedDate_EveryProblemReported(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events", map[string]any{
		"person_id": p, "event_type": "birth",
		"event_date": map[string]any{"value": "1890-13", "qualifier": "circa", "end": "1892-13"},
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	_, errs := fieldErrors(t, rec)
	for _, f := range []string{"event_date.value", "event_date.qualifier", "event_date.end"} {
		assert.NotNil(t, errorFor(errs, f), "no error for %s in %v", f, errs)
	}
}

// Scenario 4, continued: a part of the wrong type is reported once, as the
// wrong type, not also as missing.
func TestQualifiedDate_WrongTypeReportedOnce(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events", map[string]any{
		"person_id": p, "event_type": "birth",
		"event_date": map[string]any{"value": 1890, "qualifier": 5},
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	_, errs := fieldErrors(t, rec)
	for _, f := range []string{"event_date.value", "event_date.qualifier"} {
		var found []fieldError
		for _, fe := range errs {
			if fe.Field == f {
				found = append(found, fe)
			}
		}
		if assert.Len(t, found, 1, "errors for %s: %v", f, errs) {
			assert.Equal(t, "invalid_value", found[0].Code)
		}
	}
}

// Scenario 5
func TestQualifiedDate_Updates(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	le := e.createEventWith(t, c, p, map[string]any{"event_date": between("1890", "1892")})
	path := "/api/v1/cases/" + c + "/life-events/" + le["id"].(string)

	rec := e.c.DoV("PATCH", path, map[string]any{"event_place": "Palermo"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, between("1890", "1892"), decode(t, rec)["event_date"], "left out, so kept")

	rec = e.c.DoV("PATCH", path, map[string]any{"event_date": qualified("1890", "about")})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, qualified("1890", "about"), decode(t, rec)["event_date"], "replaced whole, so no end")

	rec = e.c.DoV("PATCH", path, map[string]any{"event_date": nil})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Nil(t, decode(t, rec)["event_date"], "null clears it")
}

// ---------------------------------------------------------------------------
// C. Registration date, the activity log, order, and the trash
// ---------------------------------------------------------------------------

// Scenario 6
func TestDocument_RegistrationDateReplacesRecordedDate(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	doc := e.createDocumentWith(t, c, p, map[string]any{"registration_date": "1890-03-20"})
	assert.Equal(t, "1890-03-20", doc["registration_date"])
	assert.NotContains(t, doc, "recorded_date")

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/documents", map[string]any{
		"person_id": p, "document_type": "birth_certificate", "title": "Birth certificate",
		"recorded_date": "1890-03-20",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	_, errs := fieldErrors(t, rec)
	fe := errorFor(errs, "recorded_date")
	if assert.NotNil(t, fe) {
		assert.Equal(t, "unknown_field", fe.Code)
	}
}

// Scenario 7
func TestLog_DateChanges(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	le := e.createEventWith(t, c, p, map[string]any{"event_date": qualified("1890", "about")})
	id := le["id"].(string)
	rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+id,
		map[string]any{"event_date": qualified("1890-03-15", "exact")})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []testhelpers.Change{{
		Field: "event_date",
		From:  qualified("1890", "about"),
		To:    qualified("1890-03-15", "exact"),
	}}, e.lastEntry(t, c, "life_event", id).Changes)

	doc := e.createDocumentWith(t, c, p, map[string]any{"issue_date": "1950"})
	docID := doc["id"].(string)
	rec = e.c.DoV("PATCH", "/api/v1/cases/"+c+"/documents/"+docID, map[string]any{"issue_date": "1950-07"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []testhelpers.Change{{Field: "issue_date", From: "1950", To: "1950-07"}},
		e.lastEntry(t, c, "document", docID).Changes)
}

// Scenario 9
func TestLifeEvents_OrderedByDate(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	// Created out of order, so the list order cannot come from creation time.
	undated := e.createEventWith(t, c, p, map[string]any{"event_type": "other"})
	later := e.createEventWith(t, c, p, map[string]any{"event_type": "death", "event_date": qualified("1952-11-03", "exact")})
	precise := e.createEventWith(t, c, p, map[string]any{"event_type": "other", "event_date": qualified("1890-03-15", "exact")})
	yearOnly := e.createEventWith(t, c, p, map[string]any{"event_type": "birth", "event_date": qualified("1890", "about")})

	var ids []string
	for _, item := range e.list(t, "/api/v1/cases/"+c+"/life-events") {
		ids = append(ids, item["id"].(string))
	}
	assert.Equal(t, []string{
		yearOnly["id"].(string), precise["id"].(string), later["id"].(string), undated["id"].(string),
	}, ids)
}

// Scenario 10
func TestTrash_ListsLifeEventDateAsQualified(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	le := e.createEventWith(t, c, p, map[string]any{"event_date": between("1890", "1892")})
	e.c.MustDelete(t, "/api/v1/cases/"+c+"/life-events/"+le["id"].(string))

	rec := e.c.Do("GET", "/api/v1/cases/"+c+"/trash", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	events, _ := body["life_events"].([]any)
	require.Len(t, events, 1, "trash: %v", body)
	assert.Equal(t, between("1890", "1892"), events[0].(map[string]any)["event_date"])
}
