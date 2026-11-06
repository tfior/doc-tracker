// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cases

import (
	"context"
	"time"

	"github.com/tfior/doc-tracker/internal/activitylog"
)

type Case struct {
	ID                  string    `json:"id"`
	Title               string    `json:"title"`
	Status              string    `json:"status"`
	PrimaryRootPersonID *string   `json:"primary_root_person_id"`
	Version             int       `json:"version"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ClaimLineSummary struct {
	Total            int `json:"total"`
	NotYetResearched int `json:"not_yet_researched"`
	Researching      int `json:"researching"`
	Paused           int `json:"paused"`
	Ineligible       int `json:"ineligible"`
	Eligible         int `json:"eligible"`
}

// CaseSummary is a case as the case list shows it.
type CaseSummary struct {
	Case
	Progress Progress `json:"progress"`
}

type CaseDetail struct {
	Case
	ClaimLineSummary ClaimLineSummary `json:"claim_line_summary"`
	Progress         Progress         `json:"progress"`
}

// Progress is the count of a case's Documents in the packet in each readiness
// status.
type Progress struct {
	Ready      int `json:"ready"`
	InProgress int `json:"in_progress"`
	NotStarted int `json:"not_started"`
}

// ProgressCounter counts each case's Documents in the packet by readiness.
// The documents module owns the rule; the cases module only reports it.
type ProgressCounter interface {
	Progress(ctx context.Context, caseIDs []string) (map[string]Progress, error)
}

// logFields is what the activity log compares when a case changes.
func (c *Case) logFields() activitylog.Fields {
	return activitylog.Fields{"title": c.Title, "status": c.Status}
}
