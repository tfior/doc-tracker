// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setPhases saves the given phase statuses, by label, and returns the document.
func (e *testEnv) setPhases(t *testing.T, caseID string, d map[string]any, labels map[string]string, extra map[string]any) map[string]any {
	t.Helper()
	req := map[string]any{}
	for phase, label := range labels {
		req[phase+"_status_id"] = e.statusID(t, phase, label)
	}
	for k, v := range extra {
		req[k] = v
	}
	return body(t, e.do("PATCH", docPath(caseID, d), req), http.StatusOK)
}

func readiness(t *testing.T, d map[string]any) map[string]any {
	t.Helper()
	r, ok := d["readiness"].(map[string]any)
	require.True(t, ok, "readiness is %v", d["readiness"])
	return r
}

func reasons(t *testing.T, r map[string]any) []string {
	t.Helper()
	raw, ok := r["reasons"].([]any)
	require.True(t, ok, "reasons is %v", r["reasons"])
	out := []string{}
	for _, v := range raw {
		out = append(out, v.(string))
	}
	return out
}

var allComplete = map[string]string{
	"official_copy": "Complete", "amendment": "Not Required", "apostille": "Complete", "translation": "Not Required",
}

// Scenario 1
func TestReadiness_NewDocumentNotStarted(t *testing.T) {
	env := newTestEnv(t)
	_, d := env.newDocument(t, nil)

	assert.Equal(t, true, d["needed"])
	r := readiness(t, d)
	assert.Equal(t, "not_started", r["status"])
	assert.Equal(t, "official_copy", r["waiting_on"])
	assert.True(t, timestamp(t, d["official_copy_status_since"]).Equal(timestamp(t, r["waiting_since"])))
	assert.Equal(t, []string{
		"official_copy_incomplete", "amendment_incomplete", "apostille_incomplete",
		"translation_incomplete", "details_not_checked",
	}, reasons(t, r))
}

// Scenario 2
func TestReadiness_NotNeeded(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)

	d = env.setPhases(t, caseID, d, map[string]string{"official_copy": "Not Required", "apostille": "Sent"}, nil)

	assert.Equal(t, false, d["needed"])
	assert.Contains(t, d, "readiness")
	assert.Nil(t, d["readiness"], "a document not needed is outside the packet")
}

// Scenario 3
func TestReadiness_NotForSubmission(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, map[string]any{"for_submission": false})

	assert.Equal(t, true, d["needed"])
	assert.Contains(t, d, "readiness")
	assert.Nil(t, d["readiness"], "a document not for submission is outside the packet")

	d = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{"for_submission": true}), http.StatusOK)
	assert.Equal(t, "not_started", readiness(t, d)["status"])
}

// Scenario 4
func TestReadiness_Ready(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)

	d = env.setPhases(t, caseID, d, allComplete, map[string]any{"details_checked": true})

	r := readiness(t, d)
	assert.Equal(t, "ready", r["status"])
	assert.Nil(t, r["waiting_on"])
	assert.Nil(t, r["waiting_since"])
	assert.Nil(t, r["waiting_days"])
	assert.Empty(t, reasons(t, r))
}

// Scenario 5
func TestReadiness_WaitingOnDetailsCheck(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)
	d = env.setPhases(t, caseID, d, allComplete, map[string]any{
		"official_copy_status_since": "2026-01-10T00:00:00Z",
		"amendment_status_since":     "2026-03-01T00:00:00Z",
		"apostille_status_since":     "2026-02-01T00:00:00Z",
		"translation_status_since":   "2026-01-20T00:00:00Z",
	})

	r := readiness(t, d)
	assert.Equal(t, "in_progress", r["status"])
	assert.Equal(t, "details_check", r["waiting_on"])
	assert.Equal(t, []string{"details_not_checked"}, reasons(t, r))
	assert.True(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC).Equal(timestamp(t, r["waiting_since"])),
		"waiting since the last phase entered its status")
}

// Scenario 6
func TestReadiness_AmendmentUnknown(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)

	d = env.setPhases(t, caseID, d, map[string]string{"official_copy": "Complete"}, nil)

	r := readiness(t, d)
	assert.Equal(t, "in_progress", r["status"])
	assert.Equal(t, "amendment", r["waiting_on"])
	assert.True(t, timestamp(t, d["amendment_status_since"]).Equal(timestamp(t, r["waiting_since"])))
	assert.Equal(t, []string{
		"amendment_incomplete", "apostille_incomplete", "translation_incomplete", "details_not_checked",
	}, reasons(t, r))
}

// Scenario 7
func TestReadiness_EarliestStageWins(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)

	d = env.setPhases(t, caseID, d, map[string]string{"official_copy": "Requested", "apostille": "Complete"}, nil)

	r := readiness(t, d)
	assert.Equal(t, "in_progress", r["status"])
	assert.Equal(t, "official_copy", r["waiting_on"])
	assert.Equal(t, []string{
		"official_copy_incomplete", "amendment_incomplete", "translation_incomplete", "details_not_checked",
	}, reasons(t, r))
}

// Scenario 8
func TestReadiness_ReadyForReviewIsIncomplete(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)
	labels := map[string]string{}
	for k, v := range allComplete {
		labels[k] = v
	}
	labels["translation"] = "Ready for Review"

	d = env.setPhases(t, caseID, d, labels, map[string]any{"details_checked": true})

	r := readiness(t, d)
	assert.Equal(t, "in_progress", r["status"])
	assert.Equal(t, "translation", r["waiting_on"])
	assert.Equal(t, []string{"translation_incomplete"}, reasons(t, r))
}

// Scenario 9
func TestReadiness_NotStartedIgnoresCheck(t *testing.T) {
	env := newTestEnv(t)
	_, d := env.newDocument(t, map[string]any{"details_checked": true})

	r := readiness(t, d)
	assert.Equal(t, "not_started", r["status"])
	assert.NotContains(t, reasons(t, r), "details_not_checked")
}

// Scenario 10
func TestReadiness_WaitingSinceFollowsCorrection(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)

	d = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{
		"official_copy_status_since": "2026-02-03T04:05:06Z",
	}), http.StatusOK)

	assert.True(t, time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC).Equal(timestamp(t, readiness(t, d)["waiting_since"])))
}

// Scenario 11. There is no endpoint for a single document; the list serves it.
func TestReadiness_OnEveryResponse(t *testing.T) {
	env := newTestEnv(t)
	caseID, created := env.newDocument(t, nil)
	assert.Equal(t, "not_started", readiness(t, created)["status"], "create")

	updated := body(t, env.do("PATCH", docPath(caseID, created), map[string]any{"title": "Renamed"}), http.StatusOK)
	assert.Equal(t, "not_started", readiness(t, updated)["status"], "update")
	assert.Equal(t, true, updated["needed"], "update")

	list := body(t, env.do("GET", "/api/v1/cases/"+caseID+"/documents", nil), http.StatusOK)
	items := list["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, "not_started", readiness(t, item)["status"], "list")
	assert.Equal(t, true, item["needed"], "list")
}

// Scenario 12
func TestElapsed_DaysInStatusAndWaiting(t *testing.T) {
	env := newTestEnv(t)
	caseID, d := env.newDocument(t, nil)
	tenDaysAgo := time.Now().UTC().Add(-10*24*time.Hour - time.Hour).Format(time.RFC3339)

	d = body(t, env.do("PATCH", docPath(caseID, d), map[string]any{"official_copy_status_since": tenDaysAgo}), http.StatusOK)

	assert.EqualValues(t, 10, d["official_copy_days_in_status"])
	for _, phase := range phases[1:] {
		assert.EqualValues(t, 0, d[phase+"_days_in_status"], phase)
	}
	assert.EqualValues(t, 10, readiness(t, d)["waiting_days"])
}

// Scenario 13
func TestElapsed_CopyAge(t *testing.T) {
	env := newTestEnv(t)
	issued := time.Now().UTC().AddDate(0, 0, -100).Format("2006-01-02")

	_, d := env.newDocument(t, map[string]any{"issue_date": issued})
	assert.EqualValues(t, 100, d["copy_age_days"])

	for _, partial := range []string{"2020", "2020-05"} {
		_, d = env.newDocument(t, map[string]any{"issue_date": partial})
		assert.Contains(t, d, "copy_age_days")
		assert.Nil(t, d["copy_age_days"], partial)
	}

	_, d = env.newDocument(t, nil)
	assert.Contains(t, d, "copy_age_days")
	assert.Nil(t, d["copy_age_days"], "no issue date")
}

// ---------------------------------------------------------------------------
// Case progress
// ---------------------------------------------------------------------------

func (e *testEnv) addDocument(t *testing.T, caseID, personID string, fields map[string]any) map[string]any {
	t.Helper()
	req := map[string]any{"person_id": personID, "document_type": "birth_certificate", "title": "Birth Certificate"}
	for k, v := range fields {
		req[k] = v
	}
	return body(t, e.do("POST", "/api/v1/cases/"+caseID+"/documents", req), http.StatusCreated)
}

func (e *testEnv) progress(t *testing.T, caseID string) map[string]any {
	t.Helper()
	c := body(t, e.do("GET", "/api/v1/cases/"+caseID, nil), http.StatusOK)
	p, ok := c["progress"].(map[string]any)
	require.True(t, ok, "progress is %v", c["progress"])
	return p
}

func counts(ready, inProgress, notStarted int) map[string]any {
	return map[string]any{"ready": float64(ready), "in_progress": float64(inProgress), "not_started": float64(notStarted)}
}

// Scenario 14
func TestProgress_CountsPacketDocuments(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	p := env.createPerson(t, c.ID)

	ready := env.addDocument(t, c.ID, p.ID, nil)
	env.setPhases(t, c.ID, ready, allComplete, map[string]any{"details_checked": true})
	for range 2 {
		d := env.addDocument(t, c.ID, p.ID, nil)
		env.setPhases(t, c.ID, d, map[string]string{"official_copy": "Requested"}, nil)
	}
	env.addDocument(t, c.ID, p.ID, nil) // not started
	notNeeded := env.addDocument(t, c.ID, p.ID, nil)
	env.setPhases(t, c.ID, notNeeded, map[string]string{"official_copy": "Not Required"}, nil)
	env.addDocument(t, c.ID, p.ID, map[string]any{"for_submission": false})

	assert.Equal(t, counts(1, 2, 1), env.progress(t, c.ID))
}

// Scenario 15
func TestProgress_ZerosWithoutPacketDocuments(t *testing.T) {
	env := newTestEnv(t)
	empty := env.createCase(t)
	assert.Equal(t, counts(0, 0, 0), env.progress(t, empty.ID))

	outside := env.createCase(t)
	p := env.createPerson(t, outside.ID)
	d := env.addDocument(t, outside.ID, p.ID, nil)
	env.setPhases(t, outside.ID, d, map[string]string{"official_copy": "Not Required"}, nil)
	env.addDocument(t, outside.ID, p.ID, map[string]any{"for_submission": false})
	assert.Equal(t, counts(0, 0, 0), env.progress(t, outside.ID))
}

// Scenario 16
func TestProgress_LeavesOutTrashed(t *testing.T) {
	env := newTestEnv(t)
	c := env.createCase(t)
	kept := env.createPerson(t, c.ID)
	trashedPerson := env.createPerson(t, c.ID)
	env.addDocument(t, c.ID, kept.ID, nil)
	trashedDoc := env.addDocument(t, c.ID, kept.ID, nil)
	env.addDocument(t, c.ID, trashedPerson.ID, nil)
	require.Equal(t, counts(0, 0, 3), env.progress(t, c.ID))

	docURL := docPath(c.ID, trashedDoc)
	personURL := "/api/v1/cases/" + c.ID + "/people/" + trashedPerson.ID
	env.c.MustDelete(t, docURL)
	env.c.MustDelete(t, personURL)
	assert.Equal(t, counts(0, 0, 1), env.progress(t, c.ID))

	require.Equal(t, http.StatusNoContent, env.c.Do("POST", docURL+"/restore", nil).Code)
	require.Equal(t, http.StatusNoContent, env.c.Do("POST", personURL+"/restore", nil).Code)
	assert.Equal(t, counts(0, 0, 3), env.progress(t, c.ID))
}

// Scenario 17
func TestProgress_InListAndDetail(t *testing.T) {
	env := newTestEnv(t)
	one := env.createCase(t)
	two := env.createCase(t)
	p := env.createPerson(t, one.ID)
	env.addDocument(t, one.ID, p.ID, nil)
	q := env.createPerson(t, two.ID)
	d := env.addDocument(t, two.ID, q.ID, nil)
	env.setPhases(t, two.ID, d, allComplete, map[string]any{"details_checked": true})
	env.addDocument(t, two.ID, q.ID, nil)

	assert.Equal(t, counts(0, 0, 1), env.progress(t, one.ID))
	assert.Equal(t, counts(1, 0, 1), env.progress(t, two.ID))

	list := body(t, env.do("GET", "/api/v1/cases", nil), http.StatusOK)
	byID := map[string]any{}
	for _, item := range list["items"].([]any) {
		c := item.(map[string]any)
		byID[c["id"].(string)] = c["progress"]
	}
	assert.Equal(t, counts(0, 0, 1), byID[one.ID])
	assert.Equal(t, counts(1, 0, 1), byID[two.ID])
}
