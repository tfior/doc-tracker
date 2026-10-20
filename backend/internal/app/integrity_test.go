// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the integrity rules fixed in Milestone 3: family links
// cannot form loops, documents stay with their life event's person, and the
// trash hides and freezes everything beneath a trashed record.

const missingID = "00000000-0000-0000-0000-0000000000ff"

func errorBody(t *testing.T, rec *httptest.ResponseRecorder) (code, message string) {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "response is not an error body: %s", rec.Body.String())
	return body.Error.Code, body.Error.Message
}

func (e *env) addParent(caseID, childID, parentID string) *httptest.ResponseRecorder {
	return e.c.Do("POST", "/api/v1/cases/"+caseID+"/people/"+childID+"/relationships",
		map[string]any{"parent_id": parentID})
}

func (e *env) mustAddParent(t *testing.T, caseID, childID, parentID string) {
	t.Helper()
	rec := e.addParent(caseID, childID, parentID)
	require.Equal(t, http.StatusCreated, rec.Code, "setup: add parent: %s", rec.Body.String())
}

func (e *env) createLifeEvent(t *testing.T, caseID, personID string) string {
	t.Helper()
	return e.c.MustCreate(t, "/api/v1/cases/"+caseID+"/life-events",
		map[string]any{"person_id": personID, "event_type": "birth"})
}

// createDocument creates a document for a person, under a life event when eventID is not empty.
func (e *env) createDocument(t *testing.T, caseID, personID, eventID string) string {
	t.Helper()
	body := map[string]any{"person_id": personID, "document_type": "birth_certificate", "title": "Birth certificate"}
	if eventID != "" {
		body["life_event_id"] = eventID
	}
	return e.c.MustCreate(t, "/api/v1/cases/"+caseID+"/documents", body)
}

func (e *env) trash(t *testing.T, path string) {
	t.Helper()
	e.c.MustDelete(t, path)
}

func ids(items []map[string]any) []string {
	out := []string{}
	for _, it := range items {
		out = append(out, it["id"].(string))
	}
	return out
}

func (e *env) parentIDs(t *testing.T, caseID, personID string) []string {
	t.Helper()
	p := findByID(e.list(t, "/api/v1/cases/"+caseID+"/people"), personID)
	require.NotNil(t, p, "person %s not listed", personID)
	out := []string{}
	for _, id := range p["parent_ids"].([]any) {
		out = append(out, id.(string))
	}
	return out
}

// ---------------------------------------------------------------------------
// A. Family links
// ---------------------------------------------------------------------------

// Scenario 1
func TestParentLinks_DirectLoopRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	a, b := e.createPerson(t, c, "A"), e.createPerson(t, c, "B")
	e.mustAddParent(t, c, b, a) // A is B's parent

	rec := e.addParent(c, a, b) // B as A's parent would make A their own ancestor

	assert.Equal(t, http.StatusConflict, rec.Code)
	code, _ := errorBody(t, rec)
	assert.Equal(t, "conflict", code)
	assert.Empty(t, e.parentIDs(t, c, a))
}

// Scenario 2
func TestParentLinks_LongerLoopRefused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	a, b, cc := e.createPerson(t, c, "A"), e.createPerson(t, c, "B"), e.createPerson(t, c, "C")
	e.mustAddParent(t, c, b, a)  // A -> B
	e.mustAddParent(t, c, cc, b) // B -> C

	rec := e.addParent(c, a, cc) // C as A's parent closes the loop

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Empty(t, e.parentIDs(t, c, a))
}

// Scenario 3: guards against refusing legitimate trees.
func TestParentLinks_DeepTreeAccepted(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	grandparent, parent, child, otherParent :=
		e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Antonio"),
		e.createPerson(t, c, "Carlo"), e.createPerson(t, c, "Linda")
	e.mustAddParent(t, c, child, parent)

	assert.Equal(t, http.StatusCreated, e.addParent(c, parent, grandparent).Code, "grandparent above an existing parent")
	assert.Equal(t, http.StatusCreated, e.addParent(c, child, otherParent).Code, "second parent")
	assert.ElementsMatch(t, []string{parent, otherParent}, e.parentIDs(t, c, child))
}

// Scenario 4
func TestParentLinks_LimitIgnoresTrashedParent(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	child, p1, p2, p3 := e.createPerson(t, c, "Child"), e.createPerson(t, c, "P1"), e.createPerson(t, c, "P2"), e.createPerson(t, c, "P3")
	e.mustAddParent(t, c, child, p1)
	e.mustAddParent(t, c, child, p2)
	e.trash(t, "/api/v1/cases/"+c+"/people/"+p2)

	rec := e.addParent(c, child, p3)

	assert.Equal(t, http.StatusCreated, rec.Code, "the visible parent count is one, so a second may be added: %s", rec.Body.String())
	assert.ElementsMatch(t, []string{p1, p3}, e.parentIDs(t, c, child))
}

// Follows from scenario 4: the trashed parent's link still exists, so
// restoring them would give the child three parents. The restore is refused.
func TestParentLinks_RestoringWouldExceedLimit_Refused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	child, p1, p2, p3 := e.createPerson(t, c, "Child"), e.createPerson(t, c, "P1"), e.createPerson(t, c, "P2"), e.createPerson(t, c, "P3")
	e.mustAddParent(t, c, child, p1)
	e.mustAddParent(t, c, child, p2)
	e.trash(t, "/api/v1/cases/"+c+"/people/"+p2)
	e.mustAddParent(t, c, child, p3)

	rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people/"+p2+"/restore", nil)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.ElementsMatch(t, []string{p1, p3}, e.parentIDs(t, c, child))
}

// Scenario 5. Concurrency is not deterministic, so the race is attempted many
// times; any attempt that ends with three parents fails the test.
func TestParentLinks_SimultaneousAddsCannotExceedLimit(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 20; round++ {
		c := e.createCase(t) // a fresh case each round keeps lists small
		child, p1, p2, p3 := e.createPerson(t, c, "Child"), e.createPerson(t, c, "P1"), e.createPerson(t, c, "P2"), e.createPerson(t, c, "P3")
		e.mustAddParent(t, c, child, p1)

		var wg sync.WaitGroup
		start := make(chan struct{})
		codes := make([]int, 2)
		for i, parent := range []string{p2, p3} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				codes[i] = e.addParent(c, child, parent).Code
			}()
		}
		close(start)
		wg.Wait()

		require.Len(t, e.parentIDs(t, c, child), 2, "round %d: a person ended up with more than two parents (statuses %v)", round, codes)
		assert.ElementsMatch(t, []int{http.StatusCreated, http.StatusConflict}, codes, "round %d", round)
	}
}

// ---------------------------------------------------------------------------
// B. Documents and life events stay consistent
// ---------------------------------------------------------------------------

// Scenario 6
func TestReassignLifeEvent_MovesItsDocuments(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	a, b := e.createPerson(t, c, "Antonio"), e.createPerson(t, c, "Carlo")
	ev := e.createLifeEvent(t, c, a)
	doc := e.createDocument(t, c, a, ev)

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/life-events/"+ev+"/person", map[string]any{"person_id": b})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	d := findByID(e.list(t, "/api/v1/cases/"+c+"/documents"), doc)
	require.NotNil(t, d)
	assert.Equal(t, b, d["person_id"], "the document stayed with the old person")
	assert.Equal(t, ev, d["life_event_id"])
}

// Scenario 7
func TestDocument_LifeEventOfAnotherPerson_Refused(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	a, b := e.createPerson(t, c, "Antonio"), e.createPerson(t, c, "Carlo")
	evB := e.createLifeEvent(t, c, b)

	t.Run("create", func(t *testing.T) {
		rec := e.c.Do("POST", "/api/v1/cases/"+c+"/documents", map[string]any{
			"person_id": a, "life_event_id": evB, "document_type": "birth_certificate", "title": "Mismatch",
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		code, _ := errorBody(t, rec)
		assert.Equal(t, "invalid_input", code)
		assert.Empty(t, e.list(t, "/api/v1/cases/"+c+"/documents"))
	})

	t.Run("reassign", func(t *testing.T) {
		doc := e.createDocument(t, c, a, "")
		rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc+"/parent", map[string]any{
			"person_id": a, "life_event_id": evB,
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		d := findByID(e.list(t, "/api/v1/cases/"+c+"/documents"), doc)
		require.NotNil(t, d)
		assert.Nil(t, d["life_event_id"], "the mismatched life event was saved")
	})
}

// Scenario 8
func TestReassignDocument_ToAnotherPersonWithoutEvent_Ungroups(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	a, b := e.createPerson(t, c, "Antonio"), e.createPerson(t, c, "Carlo")
	evA := e.createLifeEvent(t, c, a)
	doc := e.createDocument(t, c, a, evA)

	rec := e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc+"/parent", map[string]any{"person_id": b})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	d := findByID(e.list(t, "/api/v1/cases/"+c+"/documents"), doc)
	require.NotNil(t, d)
	assert.Equal(t, b, d["person_id"])
	assert.Nil(t, d["life_event_id"], "the document kept the old person's life event")
}

// ---------------------------------------------------------------------------
// C. The trash hides and freezes what is beneath it
// ---------------------------------------------------------------------------

// Scenario 9
func TestTrashedPerson_HidesEverythingBeneath(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p, other := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Antonio")
	ev := e.createLifeEvent(t, c, p)
	docUnderEvent := e.createDocument(t, c, p, ev)
	docUnderPerson := e.createDocument(t, c, p, "")
	line := e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": p})
	otherEvent := e.createLifeEvent(t, c, other)

	e.trash(t, "/api/v1/cases/"+c+"/people/"+p)

	assert.Equal(t, []string{otherEvent}, ids(e.list(t, "/api/v1/cases/"+c+"/life-events")))
	assert.NotContains(t, ids(e.list(t, "/api/v1/cases/"+c+"/documents")), docUnderEvent)
	assert.NotContains(t, ids(e.list(t, "/api/v1/cases/"+c+"/documents")), docUnderPerson)
	assert.NotContains(t, ids(e.list(t, "/api/v1/cases/"+c+"/claim-lines")), line)
}

// Scenario 10
func TestTrashedLifeEvent_HidesItsDocuments(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	ev := e.createLifeEvent(t, c, p)
	docUnderEvent := e.createDocument(t, c, p, ev)
	docUnderPerson := e.createDocument(t, c, p, "")

	e.trash(t, "/api/v1/cases/"+c+"/life-events/"+ev)

	assert.Equal(t, []string{docUnderPerson}, ids(e.list(t, "/api/v1/cases/"+c+"/documents")))
	assert.NotContains(t, ids(e.list(t, "/api/v1/cases/"+c+"/documents")), docUnderEvent)
}

// Scenario 11
func TestLists_TrashedOrMissingCase_NotFound(t *testing.T) {
	e := newEnv(t)
	trashed := e.createCase(t)
	e.trash(t, "/api/v1/cases/"+trashed)

	for _, caseID := range []string{trashed, missingID} {
		for _, list := range []string{"people", "life-events", "documents", "claim-lines"} {
			rec := e.c.Do("GET", "/api/v1/cases/"+caseID+"/"+list, nil)
			assert.Equal(t, http.StatusNotFound, rec.Code, "GET %s of case %s", list, caseID)
			if rec.Code == http.StatusNotFound {
				code, _ := errorBody(t, rec)
				assert.Equal(t, "not_found", code)
			}
		}
	}
}

// Scenario 12
func TestWritesBeneathTrash_Refused(t *testing.T) {
	e := newEnv(t)

	// A case with a trashed person, who has a life event and documents.
	c := e.createCase(t)
	p, other := e.createPerson(t, c, "Giuseppe"), e.createPerson(t, c, "Antonio")
	ev := e.createLifeEvent(t, c, p)
	doc := e.createDocument(t, c, p, ev)
	e.trash(t, "/api/v1/cases/"+c+"/people/"+p)

	// A live person with a trashed life event.
	evOther := e.createLifeEvent(t, c, other)
	e.trash(t, "/api/v1/cases/"+c+"/life-events/"+evOther)

	// A trashed case with a person in it.
	tc := e.createCase(t)
	tp := e.createPerson(t, tc, "Carlo")
	e.trash(t, "/api/v1/cases/"+tc)

	statusID := e.anyStatusID(t, "official_copy")
	writes := []struct {
		name, method, path string
		body               any
	}{
		{"edit a document of a trashed person", "PATCH", "/api/v1/cases/" + c + "/documents/" + doc, map[string]any{"title": "x"}},
		{"change a status of that document", "PATCH", "/api/v1/cases/" + c + "/documents/" + doc + "/status", map[string]any{"phase": "official_copy", "status_id": statusID}},
		{"edit a life event of a trashed person", "PATCH", "/api/v1/cases/" + c + "/life-events/" + ev, map[string]any{"event_place": "x"}},
		{"create a life event under a trashed person", "POST", "/api/v1/cases/" + c + "/life-events", map[string]any{"person_id": p, "event_type": "death"}},
		{"create a document under a trashed person", "POST", "/api/v1/cases/" + c + "/documents", map[string]any{"person_id": p, "document_type": "other", "title": "x"}},
		{"create a document under a trashed life event", "POST", "/api/v1/cases/" + c + "/documents", map[string]any{"person_id": other, "life_event_id": evOther, "document_type": "other", "title": "x"}},
		{"give a trashed person a parent", "POST", "/api/v1/cases/" + c + "/people/" + p + "/relationships", map[string]any{"parent_id": other}},
		{"make a trashed person someone's parent", "POST", "/api/v1/cases/" + c + "/people/" + other + "/relationships", map[string]any{"parent_id": p}},
		{"create a person in a trashed case", "POST", "/api/v1/cases/" + tc + "/people", map[string]any{"first_name": "x", "last_name": "y"}},
		{"edit a person in a trashed case", "PATCH", "/api/v1/cases/" + tc + "/people/" + tp, map[string]any{"notes": "x"}},
		{"create a claim line in a trashed case", "POST", "/api/v1/cases/" + tc + "/claim-lines", map[string]any{"root_person_id": tp}},
	}
	for _, w := range writes {
		t.Run(w.name, func(t *testing.T) {
			rec := e.c.Do(w.method, w.path, w.body)
			assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		})
	}
}

// Scenario 13
func TestRestoringParent_BringsEverythingBack(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	ev := e.createLifeEvent(t, c, p)
	doc := e.createDocument(t, c, p, ev)
	e.trash(t, "/api/v1/cases/"+c+"/people/"+p)

	require.Equal(t, http.StatusNoContent, e.c.Do("POST", "/api/v1/cases/"+c+"/people/"+p+"/restore", nil).Code)

	assert.Contains(t, ids(e.list(t, "/api/v1/cases/"+c+"/life-events")), ev)
	assert.Contains(t, ids(e.list(t, "/api/v1/cases/"+c+"/documents")), doc)
	assert.Equal(t, http.StatusOK, e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc, map[string]any{"title": "Edited"}).Code)
}

// Scenario 14
func TestRestore_ParentStillTrashed_Refused(t *testing.T) {
	e := newEnv(t)

	t.Run("document whose person is trashed", func(t *testing.T) {
		c := e.createCase(t)
		p := e.createPerson(t, c, "Giuseppe")
		doc := e.createDocument(t, c, p, "")
		e.trash(t, "/api/v1/cases/"+c+"/documents/"+doc)
		e.trash(t, "/api/v1/cases/"+c+"/people/"+p)

		rec := e.c.Do("POST", "/api/v1/cases/"+c+"/documents/"+doc+"/restore", nil)

		assert.Equal(t, http.StatusConflict, rec.Code)
		code, message := errorBody(t, rec)
		assert.Equal(t, "conflict", code)
		assert.Contains(t, strings.ToLower(message), "restore", "the message should say what to restore first")
	})

	t.Run("person whose case is trashed", func(t *testing.T) {
		c := e.createCase(t)
		p := e.createPerson(t, c, "Giuseppe")
		e.trash(t, "/api/v1/cases/"+c+"/people/"+p)
		e.trash(t, "/api/v1/cases/"+c)

		rec := e.c.Do("POST", "/api/v1/cases/"+c+"/people/"+p+"/restore", nil)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

// Scenario 15. The test holds a lock on the person, as a concurrent delete
// would, and trashes them while a document is being added beneath them.
func TestTrashingPersonWhileAddingDocument_DocumentNotCreated(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")

	ctx := context.Background()
	tx, err := e.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `SELECT id FROM people WHERE id = $1 FOR UPDATE`, p)
	require.NoError(t, err)

	result := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		result <- e.c.Do("POST", "/api/v1/cases/"+c+"/documents",
			map[string]any{"person_id": p, "document_type": "birth_certificate", "title": "Racing"})
	}()

	// Wait until the request is blocked on the lock, so the trash below really
	// happens while the request is in flight.
	waitForBlockedQuery(t, e)
	_, err = tx.ExecContext(ctx, `UPDATE people SET deleted_at = NOW() WHERE id = $1`, p)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	select {
	case rec := <-result:
		assert.Equal(t, http.StatusNotFound, rec.Code, "a document was created beneath a person trashed at the same moment: %s", rec.Body.String())
	case <-time.After(10 * time.Second):
		t.Fatal("request did not finish")
	}
	var count int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE person_id = $1`, p).Scan(&count))
	assert.Zero(t, count)
}

// ---------------------------------------------------------------------------
// D. Counts
// ---------------------------------------------------------------------------

// Scenario 16
func TestClaimLineSummary_ExcludesTrashed(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p1, p2, p3 := e.createPerson(t, c, "P1"), e.createPerson(t, c, "P2"), e.createPerson(t, c, "P3")
	l1 := e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": p1})
	e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": p2})
	e.c.MustCreate(t, "/api/v1/cases/"+c+"/claim-lines", map[string]any{"root_person_id": p3})
	e.trash(t, "/api/v1/cases/"+c+"/claim-lines/"+l1) // trashed directly
	e.trash(t, "/api/v1/cases/"+c+"/people/"+p2)      // trashed through its root ancestor

	rec := e.c.Do("GET", "/api/v1/cases/"+c, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var detail struct {
		ClaimLineSummary struct {
			Total int `json:"total"`
		} `json:"claim_line_summary"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&detail))

	assert.Equal(t, 1, detail.ClaimLineSummary.Total)
	assert.Len(t, e.list(t, "/api/v1/cases/"+c+"/claim-lines"), 1)
}

// ---------------------------------------------------------------------------
// E. Permanent delete
// ---------------------------------------------------------------------------

// Scenario 18
func TestPermanentDeleteLifeEvent_DeletesItsDocuments(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	ev := e.createLifeEvent(t, c, p)
	doc := e.createDocument(t, c, p, ev)
	e.trash(t, "/api/v1/cases/"+c+"/life-events/"+ev)

	require.Equal(t, http.StatusNoContent, e.c.Do("DELETE", "/api/v1/cases/"+c+"/life-events/"+ev+"/permanent", nil).Code)

	var count int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE id = $1`, doc).Scan(&count))
	assert.Zero(t, count, "the document survived its life event's permanent deletion")
}

// waitForBlockedQuery waits until some other session in the test database is
// waiting on a lock, and fails the test if none does within five seconds.
func waitForBlockedQuery(t *testing.T, e *env) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int
		require.NoError(t, e.db.QueryRow(`
			SELECT COUNT(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting))
		if waiting > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the request never waited on the lock")
}

// anyStatusID returns the ID of some status valid for the given phase.
func (e *env) anyStatusID(t *testing.T, phase string) string {
	t.Helper()
	rec := e.c.Do("GET", "/api/v1/document-statuses?phase="+phase, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var statuses []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&statuses))
	require.NotEmpty(t, statuses)
	return statuses[0].ID
}

// Follows from the locking added for scenarios 5 and 15: two simultaneous
// edits of the same record must both succeed, one after the other. Without an
// exclusive lock taken first, they deadlock and one fails with a server error.
func TestSimultaneousEditsOfOneRecord_BothSucceed(t *testing.T) {
	e := newEnv(t)
	c := e.createCase(t)
	p := e.createPerson(t, c, "Giuseppe")
	doc := e.createDocument(t, c, p, "")
	status := e.anyStatusID(t, "official_copy")

	for round := 0; round < 10; round++ {
		var wg sync.WaitGroup
		start := make(chan struct{})
		requests := []func() int{
			func() int {
				return e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc, map[string]any{"title": "Edited"}).Code
			},
			func() int {
				return e.c.Do("PATCH", "/api/v1/cases/"+c+"/documents/"+doc+"/status",
					map[string]any{"phase": "official_copy", "status_id": status}).Code
			},
		}
		codes := make([]int, len(requests))
		for i, req := range requests {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				codes[i] = req()
			}()
		}
		close(start)
		wg.Wait()
		require.Equal(t, []int{http.StatusOK, http.StatusOK}, codes, "round %d", round)
	}
}
