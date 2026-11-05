// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package app_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover marriages, settled in Milestone 4: a marriage between two
// people in a case is one event, linked to the spouse through
// spouse_person_id, with one set of documents; a divorce is recorded on the
// marriage it ended; and spouse and divorce details belong to marriages only
// (domain-model.md, Marriages and Spouses).

// marriage creates a marriage for a person with the given fields.
func (e *env) marriage(t *testing.T, caseID, personID string, fields map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"event_type": "marriage"}
	for k, v := range fields {
		body[k] = v
	}
	return e.createEventWith(t, caseID, personID, body)
}

// couple is a case with Giuseppe, Maria, and their marriage, linked, with one
// document under it.
type couple struct {
	c, giuseppe, maria, marriage, certificate string
}

func (e *env) newCouple(t *testing.T) couple {
	t.Helper()
	c := e.createCase(t)
	g, m := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Maria")
	ev := e.marriage(t, c, g, map[string]any{"spouse_person_id": m})["id"].(string)
	doc := e.createDocument(t, c, g, ev)
	return couple{c: c, giuseppe: g, maria: m, marriage: ev, certificate: doc}
}

// ---------------------------------------------------------------------------
// A. Linking the spouse
// ---------------------------------------------------------------------------

// Scenario 1
func TestMarriage_LinkedSpouse(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	g, m := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Maria")

	ev := e.marriage(t, c, g, map[string]any{"spouse_person_id": m})
	assert.Equal(t, m, ev["spouse_person_id"])

	listed := e.get(t, c, "life-events", ev["id"].(string))
	assert.Equal(t, g, listed["person_id"])
	assert.Equal(t, m, listed["spouse_person_id"])

	entry := e.lastEntry(t, c, "life_event", ev["id"].(string))
	assert.Equal(t, "created", entry.Action)
	assert.Equal(t, "Marriage life event for Giuseppe Rossi and Maria Rossi", entry.EntityName)
}

// Scenario 2
func TestMarriage_BadSpouseRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	g := e.createPerson(t, c, "Giuseppe")
	trashed := e.createPerson(t, c, "Lucia")
	e.trash(t, "/api/v1/cases/"+c+"/people/"+trashed)
	elsewhere := e.createPerson(t, e.createCase(t), "Maria")

	for name, spouse := range map[string]string{
		"the event's own person":      g,
		"a person in another case":    elsewhere,
		"a person in the trash":       trashed,
		"a person who does not exist": "00000000-0000-0000-0000-000000000000",
	} {
		t.Run(name, func(t *testing.T) {
			rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events",
				map[string]any{"person_id": g, "event_type": "marriage", "spouse_person_id": spouse})
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			code, errs := fieldErrors(t, rec)
			assert.Equal(t, "invalid_input", code)
			if f := errorFor(errs, "spouse_person_id"); assert.NotNil(t, f, "errors: %v", errs) {
				assert.Equal(t, "invalid_value", f.Code)
			}
		})
	}

	t.Run("a malformed ID", func(t *testing.T) {
		rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events",
			map[string]any{"person_id": g, "event_type": "marriage", "spouse_person_id": "not-an-id"})
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		code, errs := fieldErrors(t, rec)
		assert.Equal(t, "invalid_input", code)
		if f := errorFor(errs, "spouse_person_id"); assert.NotNil(t, f, "errors: %v", errs) {
			assert.Equal(t, "invalid_id", f.Code)
		}
	})
}

// Scenario 3
func TestMarriage_LinkAndFlatFieldsTogetherRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	g, m := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Maria")

	t.Run("both in one request", func(t *testing.T) {
		rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events", map[string]any{
			"person_id": g, "event_type": "marriage", "spouse_person_id": m, "spouse_name": "Maria Bianchi",
		})
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		_, errs := fieldErrors(t, rec)
		assert.NotNil(t, errorFor(errs, "spouse_name"), "errors: %v", errs)
	})

	flat := e.marriage(t, c, g, map[string]any{
		"spouse_name": "Maria Bianchi", "spouse_birth_place": "Palermo",
	})["id"].(string)

	t.Run("a link while flat fields are stored", func(t *testing.T) {
		rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+flat, map[string]any{"spouse_person_id": m})
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		_, errs := fieldErrors(t, rec)
		assert.NotNil(t, errorFor(errs, "spouse_name"), "errors: %v", errs)
		assert.NotNil(t, errorFor(errs, "spouse_birth_place"), "errors: %v", errs)
		assert.Nil(t, errorFor(errs, "spouse_birth_date"), "not stored, so not at fault")
	})

	t.Run("linking and clearing in one request: the manual promotion", func(t *testing.T) {
		rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+flat, map[string]any{
			"spouse_person_id": m, "spouse_name": nil, "spouse_birth_date": nil, "spouse_birth_place": nil,
		})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		got := decode(t, rec)
		assert.Equal(t, m, got["spouse_person_id"])
		assert.Nil(t, got["spouse_name"])
		assert.Nil(t, got["spouse_birth_place"])
	})

	t.Run("flat fields while a link is stored", func(t *testing.T) {
		rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+flat, map[string]any{"spouse_name": "Maria Bianchi"})
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		_, errs := fieldErrors(t, rec)
		assert.NotNil(t, errorFor(errs, "spouse_name"), "errors: %v", errs)
	})
}

// Scenario 4
func TestMarriage_SpouseAndDivorceOnlyOnMarriages(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	g, m := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Maria")

	for field, value := range map[string]any{
		"spouse_person_id":   m,
		"spouse_name":        "Maria Bianchi",
		"spouse_birth_date":  qualified("1880", "about"),
		"spouse_birth_place": "Palermo",
		"divorce_date":       qualified("1920", "exact"),
		"divorce_place":      "Reno",
	} {
		t.Run("create with "+field, func(t *testing.T) {
			rec := e.c.Do("POST", "/api/v1/cases/"+c+"/life-events",
				map[string]any{"person_id": g, "event_type": "death", field: value})
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			_, errs := fieldErrors(t, rec)
			if f := errorFor(errs, field); assert.NotNil(t, f, "errors: %v", errs) {
				assert.Equal(t, "invalid_value", f.Code)
			}
		})
	}

	ev := e.marriage(t, c, g, map[string]any{"spouse_name": "Maria Bianchi", "divorce_place": "Reno"})["id"].(string)

	t.Run("changing the type away from marriage while they hold values", func(t *testing.T) {
		rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+ev, map[string]any{"event_type": "other"})
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		_, errs := fieldErrors(t, rec)
		assert.NotNil(t, errorFor(errs, "spouse_name"), "errors: %v", errs)
		assert.NotNil(t, errorFor(errs, "divorce_place"), "errors: %v", errs)
		assert.Equal(t, "marriage", e.get(t, c, "life-events", ev)["event_type"], "nothing saved")
	})

	t.Run("changing the type and clearing them in one request", func(t *testing.T) {
		rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+ev,
			map[string]any{"event_type": "other", "spouse_name": nil, "divorce_place": nil})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "other", decode(t, rec)["event_type"])
	})
}

// Scenario 5
func TestMarriage_Divorce(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	g := e.createPerson(t, c, "Giuseppe")

	ev := e.marriage(t, c, g, map[string]any{"divorce_date": between("1919", "1921"), "divorce_place": "Reno"})
	assert.Equal(t, between("1919", "1921"), ev["divorce_date"])
	assert.Equal(t, "Reno", ev["divorce_place"])
	id := ev["id"].(string)

	rec := e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+id,
		map[string]any{"divorce_date": qualified("1919-13", "exact")})
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	_, errs := fieldErrors(t, rec)
	assert.NotNil(t, errorFor(errs, "divorce_date.value"), "errors: %v", errs)

	rec = e.c.DoV("PATCH", "/api/v1/cases/"+c+"/life-events/"+id, map[string]any{"divorce_place": "Las Vegas"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	entry := e.lastEntry(t, c, "life_event", id)
	require.Len(t, entry.Changes, 1)
	assert.Equal(t, "divorce_place", entry.Changes[0].Field)
	assert.Equal(t, "Reno", entry.Changes[0].From)
	assert.Equal(t, "Las Vegas", entry.Changes[0].To)
}

// Scenario 6
func TestMarriage_Moving(t *testing.T) {
	e := newEnv(t)
	k := e.newCouple(t)

	rec := e.c.DoV("PATCH", "/api/v1/cases/"+k.c+"/life-events/"+k.marriage, map[string]any{"person_id": k.maria})
	require.Equal(t, http.StatusBadRequest, rec.Code, "moving a marriage to its own spouse: %s", rec.Body.String())
	_, errs := fieldErrors(t, rec)
	assert.NotNil(t, errorFor(errs, "person_id"), "errors: %v", errs)

	carlo := e.createPerson(t, k.c, "Carlo")
	rec = e.c.DoV("PATCH", "/api/v1/cases/"+k.c+"/life-events/"+k.marriage, map[string]any{"person_id": carlo})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, carlo, e.get(t, k.c, "documents", k.certificate)["person_id"], "its documents move with it")
}

// ---------------------------------------------------------------------------
// B. The trash
// ---------------------------------------------------------------------------

// Scenario 7
func TestMarriage_StaysWhileEitherSpouseIsActive(t *testing.T) {
	e := newEnv(t)
	k := e.newCouple(t)
	base := "/api/v1/cases/" + k.c
	own := e.createDocument(t, k.c, k.giuseppe, "") // goes with Giuseppe

	assert.Equal(t, map[string]any{"life_events": 0.0, "documents": 1.0, "claim_lines": 0.0},
		e.impact(t, base+"/people/"+k.giuseppe), "the marriage stays with Maria, so it does not go with Giuseppe")

	e.trash(t, base+"/people/"+k.giuseppe)

	assert.NotNil(t, findByID(e.list(t, base+"/life-events"), k.marriage), "the marriage is still listed")
	assert.NotNil(t, findByID(e.list(t, base+"/documents"), k.certificate), "its document is still listed")
	assert.Nil(t, findByID(e.list(t, base+"/documents"), own), "Giuseppe's own document is hidden")
	assert.Equal(t, http.StatusOK, e.c.DoV("PATCH", base+"/life-events/"+k.marriage, map[string]any{"event_place": "Palermo"}).Code)
	assert.Equal(t, http.StatusOK, e.c.DoV("PATCH", base+"/documents/"+k.certificate, map[string]any{"title": "Marriage certificate"}).Code)
	added := e.c.Do("POST", base+"/documents", map[string]any{
		"person_id": k.giuseppe, "life_event_id": k.marriage, "document_type": "marriage_certificate", "title": "Second copy",
	})
	assert.Equal(t, http.StatusCreated, added.Code, "a document can still be added under the marriage: %s", added.Body.String())

	assert.Equal(t, map[string]any{"life_events": 1.0, "documents": 2.0, "claim_lines": 0.0},
		e.impact(t, base+"/people/"+k.maria), "with Giuseppe gone, the marriage and its documents go with Maria")

	e.trash(t, base+"/people/"+k.maria)

	assert.Nil(t, findByID(e.list(t, base+"/life-events"), k.marriage), "both spouses trashed: hidden")
	assert.Nil(t, findByID(e.list(t, base+"/documents"), k.certificate))
	assert.Equal(t, http.StatusNotFound, e.c.DoV("PATCH", base+"/life-events/"+k.marriage, map[string]any{"event_place": "x"}).Code, "and frozen")
	assert.Equal(t, http.StatusNotFound, e.c.DoV("PATCH", base+"/documents/"+k.certificate, map[string]any{"title": "x"}).Code)

	require.Equal(t, http.StatusNoContent, e.c.Do("POST", base+"/people/"+k.maria+"/restore", nil).Code)

	assert.NotNil(t, findByID(e.list(t, base+"/life-events"), k.marriage), "restoring either spouse brings it back")
	assert.NotNil(t, findByID(e.list(t, base+"/documents"), k.certificate))
}

// Scenario 8
func TestMarriage_RestoreWhileEitherSpouseIsActive(t *testing.T) {
	e := newEnv(t)
	k := e.newCouple(t)
	base := "/api/v1/cases/" + k.c
	e.trash(t, base+"/life-events/"+k.marriage)
	e.trash(t, base+"/people/"+k.giuseppe)
	e.trash(t, base+"/people/"+k.maria)

	rec := e.c.Do("POST", base+"/life-events/"+k.marriage+"/restore", nil)
	assertConflict(t, rec, "both spouses in the trash")

	require.Equal(t, http.StatusNoContent, e.c.Do("POST", base+"/people/"+k.maria+"/restore", nil).Code)
	rec = e.c.Do("POST", base+"/life-events/"+k.marriage+"/restore", nil)
	require.Equal(t, http.StatusNoContent, rec.Code, "Maria is active: %s", rec.Body.String())
	assert.NotNil(t, findByID(e.list(t, base+"/life-events"), k.marriage))
}

// ---------------------------------------------------------------------------
// C. Permanent deletion
// ---------------------------------------------------------------------------

// Scenario 9
func TestMarriage_PermanentlyDeletingTheLinkedSpouse(t *testing.T) {
	e := newEnv(t)
	k := e.newCouple(t)
	base := "/api/v1/cases/" + k.c
	e.patchPerson(t, k.c, k.maria, map[string]any{
		"last_name": "Bianchi", "birth_date": qualified("1880", "about"), "birth_place": "Palermo",
	})
	e.trash(t, base+"/people/"+k.maria)

	assert.Equal(t, map[string]any{"life_events": 1.0, "documents": 0.0, "claim_lines": 0.0},
		e.impact(t, base+"/people/"+k.maria), "her birth event goes; the marriage does not")

	e.permanentlyDelete(t, base+"/people/"+k.maria)

	ev := e.get(t, k.c, "life-events", k.marriage)
	assert.Nil(t, ev["spouse_person_id"])
	assert.Equal(t, "Maria Bianchi", ev["spouse_name"])
	assert.Equal(t, qualified("1880", "about"), ev["spouse_birth_date"])
	assert.Equal(t, "Palermo", ev["spouse_birth_place"])
	assert.NotNil(t, findByID(e.list(t, base+"/documents"), k.certificate), "its document remains")

	entry := e.lastEntry(t, k.c, "life_event", k.marriage)
	assert.Equal(t, "updated", entry.Action)
	assert.NotContains(t, e.permanentlyDeleted(t, k.c, "life_event"), k.marriage)
}

// Scenario 10
func TestMarriage_PermanentlyDeletingTheEventsPerson(t *testing.T) {
	e := newEnv(t)
	k := e.newCouple(t)
	base := "/api/v1/cases/" + k.c
	e.patchPerson(t, k.c, k.giuseppe, map[string]any{"birth_place": "Naples"})
	e.trash(t, base+"/people/"+k.giuseppe)

	assert.Equal(t, map[string]any{"life_events": 1.0, "documents": 0.0, "claim_lines": 0.0},
		e.impact(t, base+"/people/"+k.giuseppe), "his birth event goes; the marriage passes to Maria")

	e.permanentlyDelete(t, base+"/people/"+k.giuseppe)

	ev := e.get(t, k.c, "life-events", k.marriage)
	assert.Equal(t, k.maria, ev["person_id"], "the surviving spouse takes the marriage")
	assert.Nil(t, ev["spouse_person_id"])
	assert.Equal(t, "Giuseppe Rossi", ev["spouse_name"])
	assert.Equal(t, "Naples", ev["spouse_birth_place"])
	assert.Equal(t, k.maria, e.get(t, k.c, "documents", k.certificate)["person_id"], "the documents go with it")

	assert.NotContains(t, e.permanentlyDeleted(t, k.c, "life_event"), k.marriage)
	assert.NotContains(t, e.permanentlyDeleted(t, k.c, "document"), k.certificate)
}

// Scenario 11
func TestMarriage_WithoutLinkedSpouseGoesWithItsPerson(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	base := "/api/v1/cases/" + c
	g := e.createPerson(t, c, "Giuseppe")
	ev := e.marriage(t, c, g, map[string]any{"spouse_name": "Maria Bianchi"})["id"].(string)
	doc := e.createDocument(t, c, g, ev)
	e.trash(t, base+"/people/"+g)

	e.permanentlyDelete(t, base+"/people/"+g)

	assert.Contains(t, e.permanentlyDeleted(t, c, "life_event"), ev)
	assert.Contains(t, e.permanentlyDeleted(t, c, "document"), doc)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM life_events WHERE id = $1`, ev).Scan(&n))
	assert.Zero(t, n)
}
