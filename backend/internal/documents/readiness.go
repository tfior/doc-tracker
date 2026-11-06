// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents

import (
	"time"

	"github.com/tfior/doc-tracker/platform"
)

// The readiness statuses of a Document in the packet (domain rule 35).
const (
	ReadinessReady      = "ready"
	ReadinessInProgress = "in_progress"
	ReadinessNotStarted = "not_started"
)

// detailsCheckStage is the stage a Document waits on once every phase is
// complete but its details have not been checked.
const detailsCheckStage = "details_check"

// Readiness says whether a Document in the packet is ready to be submitted,
// and if not, what it is waiting on and why.
type Readiness struct {
	Status       string     `json:"status"`
	WaitingOn    *string    `json:"waiting_on"`
	WaitingSince *time.Time `json:"waiting_since"`
	WaitingDays  *int       `json:"waiting_days"`
	Reasons      []string   `json:"reasons"`
}

// Progress is the count of a Case's Documents in the packet in each
// readiness status (domain rule 38).
type Progress struct {
	Ready      int `json:"ready"`
	InProgress int `json:"in_progress"`
	NotStarted int `json:"not_started"`
}

func (p *Progress) add(status string) {
	switch status {
	case ReadinessReady:
		p.Ready++
	case ReadinessInProgress:
		p.InProgress++
	case ReadinessNotStarted:
		p.NotStarted++
	}
}

// phaseNames are the phases in the order a Document waits on them (domain
// rule 39).
var phaseNames = [4]string{"official_copy", "amendment", "apostille", "translation"}

// packetState is what readiness status needs to know about a Document.
// Buckets are in the order of phaseNames.
type packetState struct {
	needed        bool
	forSubmission bool
	checked       bool
	buckets       [4]string
}

// statusOf is a Document's readiness status (domain rule 35), and false for a
// Document outside the packet, which has none.
func statusOf(s packetState) (string, bool) {
	if !s.needed || !s.forSubmission {
		return "", false
	}
	allComplete, allNotStarted := true, true
	for _, b := range s.buckets {
		allComplete = allComplete && b == "complete"
		allNotStarted = allNotStarted && b == "not_started"
	}
	switch {
	case allNotStarted:
		// A check made before any work began does not move the document on.
		return ReadinessNotStarted, true
	case allComplete && s.checked:
		return ReadinessReady, true
	default:
		return ReadinessInProgress, true
	}
}

// readinessOf is a Document's readiness with what it is waiting on and why
// (domain rules 35, 36 and 39), given when each phase entered its status. A
// Document outside the packet has none.
func readinessOf(s packetState, since [4]time.Time, now time.Time) *Readiness {
	status, ok := statusOf(s)
	if !ok {
		return nil
	}
	r := &Readiness{Status: status, Reasons: []string{}}
	if status == ReadinessReady {
		return r
	}

	stage, waitingSince := "", time.Time{}
	for i, b := range s.buckets {
		if b != "complete" {
			r.Reasons = append(r.Reasons, phaseNames[i]+"_incomplete")
			if stage == "" {
				stage, waitingSince = phaseNames[i], since[i]
			}
		}
	}
	if !s.checked {
		r.Reasons = append(r.Reasons, "details_not_checked")
	}
	if stage == "" {
		// Every phase is complete, so the check is what remains, since the
		// last phase entered its status.
		stage = detailsCheckStage
		for _, t := range since {
			if t.After(waitingSince) {
				waitingSince = t
			}
		}
	}
	days := wholeDays(waitingSince, now)
	r.WaitingOn, r.WaitingSince, r.WaitingDays = &stage, &waitingSince, &days
	return r
}

// wholeDays is the number of whole days from since to now, never negative.
func wholeDays(since, now time.Time) int {
	if d := now.Sub(since); d > 0 {
		return int(d / (24 * time.Hour))
	}
	return 0
}

// copyAgeDays is the age of a copy issued on issueDate, in days, when that is
// a full date (domain rule 41). It is counted in UTC, which is accurate
// enough for judging a copy's age.
func copyAgeDays(issueDate *string, now time.Time) *int {
	if issueDate == nil {
		return nil
	}
	d, ok := platform.ParsePartialDate(*issueDate)
	if !ok || d.Day == 0 {
		return nil
	}
	u := now.UTC()
	today := time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
	// Unix seconds rather than a Duration, which cannot span old dates.
	// An issue date in the future is a typing error; it reads as a new copy.
	days := max(0, int((today.Unix()-d.Earliest().Unix())/86400))
	return &days
}

// notRequiredKey is the system key of "Not Required", which on the official
// copy phase means a Document is not needed (domain rule 29).
const notRequiredKey = "not_required"

// packetStateOf reads what readiness needs from a Document's stored fields.
func packetStateOf(officialCopyKey string, forSubmission, checked bool, buckets [4]string) packetState {
	return packetState{
		needed:        officialCopyKey != notRequiredKey,
		forSubmission: forSubmission,
		checked:       checked,
		buckets:       buckets,
	}
}

// derive sets the values derived from a Document's stored fields.
func (d *Document) derive(now time.Time) {
	s := packetStateOf(d.officialCopyKey, d.ForSubmission, d.DetailsChecked, [4]string{
		d.OfficialCopyStatus.ProgressBucket, d.AmendmentStatus.ProgressBucket,
		d.ApostilleStatus.ProgressBucket, d.TranslationStatus.ProgressBucket,
	})
	d.Needed = s.needed
	d.Readiness = readinessOf(s, [4]time.Time{
		d.OfficialCopyStatusSince, d.AmendmentStatusSince, d.ApostilleStatusSince, d.TranslationStatusSince,
	}, now)
	d.OfficialCopyDaysInStatus = wholeDays(d.OfficialCopyStatusSince, now)
	d.AmendmentDaysInStatus = wholeDays(d.AmendmentStatusSince, now)
	d.ApostilleDaysInStatus = wholeDays(d.ApostilleStatusSince, now)
	d.TranslationDaysInStatus = wholeDays(d.TranslationStatusSince, now)
	d.CopyAgeDays = copyAgeDays(d.IssueDate, now)
}
