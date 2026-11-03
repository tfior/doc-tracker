// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/testhelpers"
)

// These tests cover a person's birth and death, settled in Milestone 4: they
// live on the person's birth and death life events, which the person API
// reads and writes, and a person has at most one of each (domain-model.md,
// Person and rules 6 and 7).

// eventsOfType returns the person's listed life events of one type.
func (e *env) eventsOfType(t *testing.T, caseID, personID, eventType string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, le := range e.list(t, "/api/v1/cases/"+caseID+"/life-events") {
		if le["person_id"] == personID && le["event_type"] == eventType {
			out = append(out, le)
		}
	}
	return out
}

// onlyEvent returns the person's one listed life event of a type.
func (e *env) onlyEvent(t *testing.T, caseID, personID, eventType string) map[string]any {
	t.Helper()
	events := e.eventsOfType(t, caseID, personID, eventType)
	require.Len(t, events, 1, "%s events", eventType)
	return events[0]
}

func (e *env) personVersion(t *testing.T, caseID, personID string) int {
	t.Helper()
	return version(t, e.get(t, caseID, "people", personID))
}

func (e *env) patchPerson(t *testing.T, caseID, personID string, body map[string]any) map[string]any {
	t.Helper()
	rec := e.c.DoV("PATCH", "/api/v1/cases/"+caseID+"/people/"+personID, body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return decode(t, rec)
}

func assertConflict(t *testing.T, rec *httptest.ResponseRecorder, what string) {
	t.Helper()
	assert.Equal(t, http.StatusConflict, rec.Code, "%s: %s", what, rec.Body.String())
}

// ---------------------------------------------------------------------------
// A. The person API reads and writes the birth and death events
// ---------------------------------------------------------------------------

// Scenario 1
func TestPerson_CreateWithBirthAndDeath(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people", map[string]any{
		"first_name": "Giuseppe", "last_name": "Rossi",
		"birth_date": qualified("1875-03-15", "exact"), "birth_place": "Palermo",
		"death_date": qualified("1952", "about"), "death_place": "New York",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	p := decode(t, rec)
	assert.Equal(t, qualified("1875-03-15", "exact"), p["birth_date"])
	assert.Equal(t, "Palermo", p["birth_place"])
	assert.Equal(t, qualified("1952", "about"), p["death_date"])
	assert.Equal(t, "New York", p["death_place"])

	id := p["id"].(string)
	birth := e.onlyEvent(t, c, id, "birth")
	assert.Equal(t, qualified("1875-03-15", "exact"), birth["event_date"])
	assert.Equal(t, "Palermo", birth["event_place"])
	death := e.onlyEvent(t, c, id, "death")
	assert.Equal(t, qualified("1952", "about"), death["event_date"])
	assert.Equal(t, "New York", death["event_place"])

	listed := e.get(t, c, "people", id)
	assert.Equal(t, qualified("1875-03-15", "exact"), listed["birth_date"], "as listed")
	assert.Equal(t, "New York", listed["death_place"], "as listed")
}

// Scenario 2
func TestPerson_CreateWithoutBirthOrDeathMakesNoEvents(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	e.createPerson(t, c, "Giuseppe")

	assert.Empty(t, e.list(t, "/api/v1/cases/"+c+"/life-events"))
}

// Scenario 3
func TestPerson_UpdateChangesTheExistingEvent(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	birth := e.createEventWith(t, c, p, map[string]any{"event_date": qualified("1875", "about"), "notes": "From the census"})
	personBefore := e.personVersion(t, c, p)

	got := e.patchPerson(t, c, p, map[string]any{"birth_date": qualified("1875-03-15", "exact")})
	assert.Equal(t, qualified("1875-03-15", "exact"), got["birth_date"])

	after := e.onlyEvent(t, c, p, "birth")
	assert.Equal(t, birth["id"], after["id"], "the same event, not a new one")
	assert.Equal(t, qualified("1875-03-15", "exact"), after["event_date"])
	assert.Equal(t, "From the census", after["notes"], "the rest of the event is kept")
	assert.Greater(t, version(t, after), version(t, birth), "event version")
	assert.Greater(t, e.personVersion(t, c, p), personBefore, "person version")
}

// Scenario 4
func TestPerson_UpdateCreatesMissingEvent(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	e.patchPerson(t, c, p, map[string]any{"death_place": "New York"})

	death := e.onlyEvent(t, c, p, "death")
	assert.Equal(t, "New York", death["event_place"])
	assert.Nil(t, death["event_date"])
	assert.Empty(t, e.eventsOfType(t, c, p, "birth"), "birth was not mentioned, so no birth event")
}

// Scenario 5
func TestPerson_ClearingLeavesTheEvent(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	e.patchPerson(t, c, p, map[string]any{"birth_date": qualified("1875", "exact"), "birth_place": "Palermo"})
	birth := e.onlyEvent(t, c, p, "birth")
	doc := e.createDocumentWith(t, c, p, map[string]any{"life_event_id": birth["id"]})

	got := e.patchPerson(t, c, p, map[string]any{"birth_date": nil, "birth_place": nil})
	assert.Nil(t, got["birth_date"])
	assert.Nil(t, got["birth_place"])

	after := e.onlyEvent(t, c, p, "birth")
	assert.Equal(t, birth["id"], after["id"])
	assert.Nil(t, after["event_date"])
	assert.Nil(t, after["event_place"])
	assert.Equal(t, birth["id"], e.get(t, c, "documents", doc["id"].(string))["life_event_id"], "document still attached")
}

// Scenario 6
func TestPerson_EditingTheEventChangesThePerson(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	e.patchPerson(t, c, p, map[string]any{"birth_date": qualified("1875", "about")})
	stale := e.personVersion(t, c, p)
	birth := e.onlyEvent(t, c, p, "birth")

	rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+birth["id"].(string),
		map[string]any{"event_date": qualified("1875-03-15", "exact")})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	person := e.get(t, c, "people", p)
	assert.Equal(t, qualified("1875-03-15", "exact"), person["birth_date"])
	assert.Greater(t, version(t, person), stale)

	rec = e.c.Do("PATCH", "/api/v1/cases/"+c+"/people/"+p,
		map[string]any{"version": stale, "birth_date": qualified("1876", "about")})
	assert.Equal(t, http.StatusConflict, rec.Code)
	code, _ := errorBody(t, rec)
	assert.Equal(t, "version_conflict", code)
	assert.Equal(t, qualified("1875-03-15", "exact"), e.onlyEvent(t, c, p, "birth")["event_date"], "nothing saved")
}

// ---------------------------------------------------------------------------
// B. At most one birth event and one death event
// ---------------------------------------------------------------------------

// Scenario 7
func TestPerson_AtMostOneBirthAndDeath(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	other := e.createPerson(t, c, "Antonio")
	for _, eventType := range []string{"birth", "death"} {
		t.Run(eventType, func(t *testing.T) {
			first := e.createEventWith(t, c, p, map[string]any{"event_type": eventType})

			rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events", map[string]any{"person_id": p, "event_type": eventType})
			assertConflict(t, rec, "a second "+eventType)
			_, msg := errorBody(t, rec)
			assert.Contains(t, msg, "Giuseppe Rossi", "the refusal names the person")

			other1 := e.createEventWith(t, c, p, map[string]any{"event_type": "other"})
			rec = e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+other1["id"].(string), map[string]any{"event_type": eventType})
			assertConflict(t, rec, "changing another event's type to "+eventType)

			second := e.createEventWith(t, c, other, map[string]any{"event_type": eventType})
			rec = e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+second["id"].(string), map[string]any{"person_id": p})
			assertConflict(t, rec, "moving a "+eventType+" event to someone who has one")

			// Trash the first, give the person a new one, then try to restore the first.
			e.c.MustDelete(t, "/api/v1/cases/"+c+"/life-events/"+first["id"].(string))
			replacement := e.createEventWith(t, c, p, map[string]any{"event_type": eventType})
			rec = e.c.Do("POST", "/api/v1/cases/"+c+"/life-events/"+first["id"].(string)+"/restore", nil)
			assertConflict(t, rec, "restoring a "+eventType+" event while another is active")

			e.c.MustDelete(t, "/api/v1/cases/"+c+"/life-events/"+replacement["id"].(string))
			rec = e.c.Do("POST", "/api/v1/cases/"+c+"/life-events/"+first["id"].(string)+"/restore", nil)
			assert.Equal(t, http.StatusNoContent, rec.Code, "restore with no other active: %s", rec.Body.String())
		})
	}
}

// Scenario 8
func TestPerson_TrashedBirthEventIsNotTheirs(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	e.patchPerson(t, c, p, map[string]any{"birth_date": qualified("1875", "about")})
	old := e.onlyEvent(t, c, p, "birth")
	e.c.MustDelete(t, "/api/v1/cases/"+c+"/life-events/"+old["id"].(string))

	assert.Nil(t, e.get(t, c, "people", p)["birth_date"])

	e.patchPerson(t, c, p, map[string]any{"birth_date": qualified("1875-03-15", "exact")})
	birth := e.onlyEvent(t, c, p, "birth")
	assert.NotEqual(t, old["id"], birth["id"], "a new event")
	assert.Equal(t, qualified("1875-03-15", "exact"), birth["event_date"])
}

// ---------------------------------------------------------------------------
// C. Versions, bad dates, the activity log, and order
// ---------------------------------------------------------------------------

// Scenario 9
func TestPerson_VersionRisesWithBirthAndDeathEvents(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	events := "/api/v1/cases/" + c + "/life-events/"

	rises := func(t *testing.T, what string, change func()) {
		t.Helper()
		before := e.personVersion(t, c, p)
		change()
		assert.Greater(t, e.personVersion(t, c, p), before, what)
	}
	var birthID string
	rises(t, "create", func() { birthID = e.createEventWith(t, c, p, nil)["id"].(string) })
	rises(t, "edit", func() {
		rec := e.c.DoV("PATCH", events+birthID, map[string]any{"event_place": "Palermo"})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})
	rises(t, "trash", func() { e.c.MustDelete(t, events+birthID) })
	rises(t, "restore", func() {
		rec := e.c.Do("POST", events+birthID+"/restore", nil)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	})

	var deathID string
	rises(t, "create death", func() { deathID = e.createEventWith(t, c, p, map[string]any{"event_type": "death"})["id"].(string) })
	rises(t, "change death to another type", func() {
		rec := e.c.DoV("PATCH", events+deathID, map[string]any{"event_type": "other"})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})

	marriage := e.createEventWith(t, c, p, map[string]any{"event_type": "marriage"})
	before := e.personVersion(t, c, p)
	rec := e.c.DoV("PATCH", events+marriage["id"].(string), map[string]any{"event_place": "Naples"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, before, e.personVersion(t, c, p), "a marriage is not part of the person")
}

// Scenario 9, continued: moving a birth event changes both people.
func TestPerson_MovingBirthEventChangesBothPeople(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	from := e.createPerson(t, c, "Giuseppe")
	to := e.createPerson(t, c, "Antonio")
	birth := e.createEventWith(t, c, from, nil)
	fromBefore, toBefore := e.personVersion(t, c, from), e.personVersion(t, c, to)

	rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+birth["id"].(string), map[string]any{"person_id": to})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Greater(t, e.personVersion(t, c, from), fromBefore, "the person it left")
	assert.Greater(t, e.personVersion(t, c, to), toBefore, "the person it joined")
}

// Scenario 10
func TestPerson_BadBirthDateRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people", map[string]any{
		"first_name": "Giuseppe", "last_name": "Rossi",
		"birth_date": qualified("1875-02-30", "exact"),
		"death_date": map[string]any{"value": "1952"},
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	_, errs := fieldErrors(t, rec)
	if fe := errorFor(errs, "birth_date.value"); assert.NotNil(t, fe, "%v", errs) {
		assert.Equal(t, "invalid_date", fe.Code)
	}
	if fe := errorFor(errs, "death_date.qualifier"); assert.NotNil(t, fe, "%v", errs) {
		assert.Equal(t, "required", fe.Code)
	}
	assert.Empty(t, e.list(t, "/api/v1/cases/"+c+"/people"))
	assert.Empty(t, e.list(t, "/api/v1/cases/"+c+"/life-events"))
}

// Scenario 10, continued: if the event cannot be written, the person is not saved either.
func TestPerson_EventWriteFails_NothingSaved(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	testhelpers.FailInserts(t, e.db, "life_events")

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people", map[string]any{
		"first_name": "Giuseppe", "last_name": "Rossi", "birth_place": "Palermo",
	})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Empty(t, e.list(t, "/api/v1/cases/"+c+"/people"), "person saved although their birth event failed")

	p := e.createPerson(t, c, "Antonio")
	rec = e.c.DoV("PATCH", "/api/v1/cases/"+c+"/people/"+p, map[string]any{"first_name": "Anthony", "birth_place": "Palermo"})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "Antonio", e.get(t, c, "people", p)["first_name"], "person changed although their birth event failed")
}

// Scenario 11
func TestLog_BirthChangeLoggedOnTheEvent(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	e.patchPerson(t, c, p, map[string]any{"birth_date": qualified("1875", "about")})
	birth := e.onlyEvent(t, c, p, "birth")
	assert.Equal(t, "created", e.lastEntry(t, c, "life_event", birth["id"].(string)).Action)
	entries := len(testhelpers.ActivityLog(t, e.db, c))

	e.patchPerson(t, c, p, map[string]any{"birth_date": qualified("1875-03-15", "exact")})

	log := testhelpers.ActivityLog(t, e.db, c)
	require.Len(t, log, entries+1, "one entry, for the event")
	entry := log[len(log)-1]
	assert.Equal(t, "life_event", entry.EntityType)
	assert.Equal(t, birth["id"], entry.EntityID)
	assert.Equal(t, "updated", entry.Action)
	assert.Equal(t, []testhelpers.Change{{
		Field: "event_date", From: qualified("1875", "about"), To: qualified("1875-03-15", "exact"),
	}}, entry.Changes)
}

// Scenario 12
func TestPeople_OrderedByBirthDate(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	undated := e.createPerson(t, c, "Undated")
	younger := e.createPerson(t, c, "Younger")
	e.patchPerson(t, c, younger, map[string]any{"birth_date": qualified("1905-09-22", "exact")})
	older := e.createPerson(t, c, "Older")
	e.patchPerson(t, c, older, map[string]any{"birth_date": qualified("1875", "about")})

	var ids []string
	for _, p := range e.list(t, "/api/v1/cases/"+c+"/people") {
		ids = append(ids, p["id"].(string))
	}
	assert.Equal(t, []string{older, younger, undated}, ids)
}

// Restoring a life event that is not in the trash is not found, even when its
// person is: there is nothing to restore.
func TestRestore_EventNotInTrashNotFound(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	birth := e.createEventWith(t, c, p, nil)
	e.c.MustDelete(t, "/api/v1/cases/"+c+"/people/"+p)

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events/"+birth["id"].(string)+"/restore", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}
