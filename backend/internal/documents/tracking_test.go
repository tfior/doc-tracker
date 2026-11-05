// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/testhelpers"
)

// These tests read responses as plain JSON, so they say exactly which fields
// the API returns.

var phases = []string{"official_copy", "amendment", "apostille", "translation"}

func body(t *testing.T, rec *httptest.ResponseRecorder, want int) map[string]any {
	t.Helper()
	require.Equal(t, want, rec.Code, rec.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func timestamp(t *testing.T, v any) time.Time {
	t.Helper()
	s, ok := v.(string)
	require.True(t, ok, "expected a timestamp, got %v", v)
	ts, err := time.Parse(time.RFC3339Nano, s)
	require.NoError(t, err)
	return ts
}

// checkedBy returns the user a document's check is recorded against.
func checkedBy(t *testing.T, d map[string]any) map[string]any {
	t.Helper()
	by, ok := d["details_checked_by"].(map[string]any)
	require.True(t, ok, "details_checked_by is %v", d["details_checked_by"])
	return by
}

func (e *testEnv) newDocument(t *testing.T, fields map[string]any) (caseID string, doc map[string]any) {
	t.Helper()
	c := e.createCase(t)
	p := e.createPerson(t, c.ID)
	req := map[string]any{"person_id": p.ID, "document_type": "birth_certificate", "title": "Birth Certificate"}
	for k, v := range fields {
		req[k] = v
	}
	return c.ID, body(t, e.do("POST", "/api/v1/cases/"+c.ID+"/documents", req), http.StatusCreated)
}

func docPath(caseID string, d map[string]any) string {
	return "/api/v1/cases/" + caseID + "/documents/" + d["id"].(string)
}

// statusID returns the ID of the status with a label in a phase's picker.
func (e *testEnv) statusID(t *testing.T, phase, label string) string {
	t.Helper()
	rec := e.do("GET", "/api/v1/document-statuses?phase="+phase, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var statuses []struct{ ID, Label string }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statuses))
	for _, s := range statuses {
		if s.Label == label {
			return s.ID
		}
	}
	t.Fatalf("no status %q in phase %s", label, phase)
	return ""
}

func fieldErrors(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var out struct {
		Error struct {
			Fields []struct{ Field string } `json:"fields"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	var fields []string
	for _, f := range out.Error.Fields {
		fields = append(fields, f.Field)
	}
	return fields
}

// Scenario 1
func TestDetailsChecked_SetOnCreate(t *testing.T) {
	env := newTestEnv(t)
	start := time.Now()

	_, d := env.newDocument(t, map[string]any{"details_checked": true})
	assert.Equal(t, true, d["details_checked"])
	assert.WithinRange(t, timestamp(t, d["details_checked_at"]), start.Add(-time.Second), time.Now().Add(time.Second))
	by := checkedBy(t, d)
	assert.Equal(t, env.c.UserID, by["id"])
	assert.Equal(t, "Test User", by["name"])

	_, d = env.newDocument(t, nil)
	assert.Equal(t, false, d["details_checked"])
	assert.Nil(t, d["details_checked_at"])
	assert.Nil(t, d["details_checked_by"])
}

// Scenario 2
func TestDetailsChecked_SetOnUpdate(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)
	start := time.Now()

	d = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{"details_checked": true}), http.StatusOK)

	assert.Equal(t, true, d["details_checked"])
	assert.WithinRange(t, timestamp(t, d["details_checked_at"]), start.Add(-time.Second), time.Now().Add(time.Second))
	assert.Equal(t, env.c.UserID, checkedBy(t, d)["id"])
}

// Scenario 3
func TestDetailsChecked_KeptOnUnrelatedSave(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, map[string]any{"details_checked": true})
	other := testhelpers.NewAppClient(t, env.db)

	after := body(t, other.DoV("PATCH", docPath(caseID, d), map[string]any{
		"details_checked": true, "title": "Certified Birth Certificate",
	}), http.StatusOK)

	assert.Equal(t, true, after["details_checked"])
	assert.Equal(t, d["details_checked_at"], after["details_checked_at"], "the check keeps its time")
	assert.Equal(t, env.c.UserID, checkedBy(t, after)["id"], "and the user who made it")
}

// Scenario 4
func TestDetailsChecked_ClearedWhenUnticked(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, map[string]any{"details_checked": true})

	d = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{"details_checked": false}), http.StatusOK)

	assert.Equal(t, false, d["details_checked"])
	assert.Nil(t, d["details_checked_at"])
	assert.Nil(t, d["details_checked_by"])
}

var recordedDetails = map[string]map[string]any{
	"recorded_given_name":  {"from": "Giuseppe", "to": "Joseph"},
	"recorded_surname":     {"from": "Rossi", "to": "Russo"},
	"recorded_birth_date":  {"from": "1875-03-15", "to": "1875-03-16"},
	"recorded_birth_place": {"from": "Palermo", "to": "Naples"},
}

func recordedFrom() map[string]any {
	fields := map[string]any{"details_checked": true}
	for field, v := range recordedDetails {
		fields[field] = v["from"]
	}
	return fields
}

// Scenario 5
func TestDetailsChecked_ClearedByRecordedChange(t *testing.T) {
	env := newTestEnv(t)
	for field, v := range recordedDetails {
		t.Run(field, func(t *testing.T) {
			caseID, d := env.newDocument(t, recordedFrom())

			d = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{field: v["to"]}), http.StatusOK)

			assert.Equal(t, false, d["details_checked"])
			assert.Nil(t, d["details_checked_at"])
			assert.Nil(t, d["details_checked_by"])
		})
	}
}

// Scenario 6
func TestDetailsChecked_KeptWhenRecordedValueUnchanged(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, recordedFrom())

	after := body(t, env.do("PATCH", docPath(caseID, d), recordedFrom()), http.StatusOK)

	assert.Equal(t, true, after["details_checked"])
	assert.Equal(t, d["details_checked_at"], after["details_checked_at"])
}

// Scenario 7
func TestDetailsChecked_RecheckedWithRecordedChange(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, recordedFrom())
	other := testhelpers.NewAppClient(t, env.db)

	after := body(t, other.DoV("PATCH", docPath(caseID, d), map[string]any{
		"recorded_given_name": "Joseph", "details_checked": true,
	}), http.StatusOK)

	assert.Equal(t, true, after["details_checked"])
	assert.True(t, timestamp(t, after["details_checked_at"]).After(timestamp(t, d["details_checked_at"])),
		"a new check is recorded")
	assert.Equal(t, other.UserID, checkedBy(t, after)["id"])

	log := testhelpers.ActivityLog(t, env.db, caseID)
	assert.Contains(t, log[len(log)-1].Changes,
		testhelpers.Change{Field: "details_checked_by", From: env.c.UserID, To: other.UserID},
		"the log shows the check passing to another user")
}

// Scenario 8
func TestDetailsChecked_ClearingIsLogged(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, recordedFrom())

	body(t, env.do("PATCH", docPath(caseID, d), map[string]any{"recorded_surname": "Russo"}), http.StatusOK)

	log := testhelpers.ActivityLog(t, env.db, caseID)
	last := log[len(log)-1]
	require.Equal(t, "updated", last.Action)
	assert.Contains(t, last.Changes, testhelpers.Change{Field: "details_checked", From: true, To: false})
}

// Scenario 10
func TestPhases_StartAtCreation(t *testing.T) {
	env := newTestEnv(t)
	_, d := env.newDocument(t, nil)

	created := timestamp(t, d["created_at"])
	for _, phase := range phases {
		assert.True(t, created.Equal(timestamp(t, d[phase+"_status_since"])), "%s since is the creation time", phase)
		assert.Contains(t, d, phase+"_reference")
		assert.Nil(t, d[phase+"_reference"], phase)
	}
}

// Scenario 11
func TestPhases_StatusChangeMovesOnlyItsSince(t *testing.T) {
	env := newTestEnv(t)
	for _, phase := range phases {
		t.Run(phase, func(t *testing.T) {
			caseID, d := env.newDocument(t, nil)

			after := body(t, env.do("PATCH", docPath(caseID, d), map[string]any{
				phase + "_status_id": env.statusID(t, phase, "Complete"),
			}), http.StatusOK)

			for _, p := range phases {
				before, now := timestamp(t, d[p+"_status_since"]), timestamp(t, after[p+"_status_since"])
				if p == phase {
					assert.True(t, now.After(before), "%s since moves to now", p)
				} else {
					assert.True(t, now.Equal(before), "%s since is unchanged", p)
				}
			}
		})
	}
}

// Scenario 12
func TestPhases_SameStatusKeepsSince(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)

	req := map[string]any{}
	for _, phase := range phases {
		req[phase+"_status_id"] = d[phase+"_status"].(map[string]any)["id"]
	}
	after := body(t, env.do("PATCH", docPath(caseID, d), req), http.StatusOK)

	for _, phase := range phases {
		before, now := timestamp(t, d[phase+"_status_since"]), timestamp(t, after[phase+"_status_since"])
		assert.True(t, now.Equal(before), phase)
	}
}

// Scenario 13
func TestPhases_SinceSentIsKept(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)
	complete := env.statusID(t, "official_copy", "Complete")

	after := body(t, env.do("PATCH", docPath(caseID, d), map[string]any{
		"official_copy_status_id":    complete,
		"official_copy_status_since": "2026-01-15T05:00:00Z",
	}), http.StatusOK)
	assert.True(t, time.Date(2026, 1, 15, 5, 0, 0, 0, time.UTC).Equal(timestamp(t, after["official_copy_status_since"])),
		"a since sent with a status change is kept")

	after = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{
		"official_copy_status_since": "2025-12-01T05:00:00",
	}), http.StatusOK)
	assert.True(t, time.Date(2025, 12, 1, 5, 0, 0, 0, time.UTC).Equal(timestamp(t, after["official_copy_status_since"])),
		"a since sent alone corrects it, read as UTC without an offset")
	assert.Equal(t, complete, after["official_copy_status"].(map[string]any)["id"], "and leaves the status alone")
}

// Scenario 14
func TestPhases_SinceRefused(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)
	body(t, env.do("PATCH", docPath(caseID, d), map[string]any{"apostille_status_since": "2026-01-15T05:00:00Z"}),
		http.StatusOK) // a valid since is accepted, so the refusals below are the checks at work

	for what, value := range map[string]any{
		"in the future": time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		"null":          nil,
		"not a time":    "last Tuesday",
	} {
		t.Run(what, func(t *testing.T) {
			rec := env.do("PATCH", docPath(caseID, d), map[string]any{"apostille_status_since": value})
			assert.Equal(t, []string{"apostille_status_since"}, fieldErrors(t, rec))
		})
	}
}

// Scenario 15
func TestPhases_Reference(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)
	path := docPath(caseID, d)

	d = body(t, env.do("PATCH", path, map[string]any{"official_copy_reference": "Order 4471"}), http.StatusOK)
	assert.Equal(t, "Order 4471", d["official_copy_reference"])

	d = body(t, env.do("PATCH", path, map[string]any{
		"official_copy_status_id": env.statusID(t, "official_copy", "Requested"),
	}), http.StatusOK)
	assert.Equal(t, "Order 4471", d["official_copy_reference"], "a status change keeps the reference")

	d = body(t, env.do("PATCH", path, map[string]any{"official_copy_reference": nil}), http.StatusOK)
	assert.Nil(t, d["official_copy_reference"])
}

// Scenario 16
func TestPhases_ChangesAreLogged(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)

	after := body(t, env.do("PATCH", docPath(caseID, d), map[string]any{
		"translation_status_id": env.statusID(t, "translation", "Sent"),
		"translation_reference": "Job 12",
	}), http.StatusOK)

	log := testhelpers.ActivityLog(t, env.db, caseID)
	changes := map[string]testhelpers.Change{}
	for _, c := range log[len(log)-1].Changes {
		changes[c.Field] = c
	}
	require.Contains(t, changes, "translation_status_since", "an automatic since change is logged")
	assert.True(t, timestamp(t, after["translation_status_since"]).Equal(timestamp(t, changes["translation_status_since"].To)))
	assert.Equal(t, testhelpers.Change{Field: "translation_reference", From: nil, To: "Job 12"}, changes["translation_reference"])
}

// Scenario 18
func TestForSubmission(t *testing.T) {
	env := newTestEnv(t)

	_, d := env.newDocument(t, nil)
	assert.Equal(t, true, d["for_submission"], "defaults to true")

	caseID, d := env.newDocument(t, map[string]any{"for_submission": false})
	assert.Equal(t, false, d["for_submission"], "can be set on create")

	d = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{"for_submission": true}), http.StatusOK)
	assert.Equal(t, true, d["for_submission"], "and on update")

	log := testhelpers.ActivityLog(t, env.db, caseID)
	assert.Contains(t, log[len(log)-1].Changes, testhelpers.Change{Field: "for_submission", From: false, To: true})
}

// Scenario 19
func TestDocumentTypes_New(t *testing.T) {
	env := newTestEnv(t)
	for _, docType := range []string{"non_naturalization_certificate", "divorce_record", "court_order"} {
		t.Run(docType, func(t *testing.T) {
			_, d := env.newDocument(t, map[string]any{"document_type": docType})
			assert.Equal(t, docType, d["document_type"])

			caseID, d := env.newDocument(t, nil)
			d = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{"document_type": docType}), http.StatusOK)
			assert.Equal(t, docType, d["document_type"])
		})
	}

	caseID, d := env.newDocument(t, nil)
	rec := env.do("PATCH", docPath(caseID, d), map[string]any{"document_type": "passport"})
	assert.Equal(t, []string{"document_type"}, fieldErrors(t, rec))
}
