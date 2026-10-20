// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cases

import (
	"time"

	"github.com/tfior/doc-tracker/internal/activitylog"
)

type Case struct {
	ID                  string    `json:"id"`
	Title               string    `json:"title"`
	Status              string    `json:"status"`
	PrimaryRootPersonID *string   `json:"primary_root_person_id"`
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

type CaseDetail struct {
	Case
	ClaimLineSummary ClaimLineSummary `json:"claim_line_summary"`
}

// logFields is what the activity log compares when a case changes.
func (c *Case) logFields() activitylog.Fields {
	return activitylog.Fields{"title": c.Title, "status": c.Status}
}
