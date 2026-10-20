// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package claimlines

import (
	"time"

	"github.com/tfior/doc-tracker/internal/activitylog"
)

type ClaimLine struct {
	ID           string    `json:"id"`
	CaseID       string    `json:"case_id"`
	RootPersonID string    `json:"root_person_id"`
	Status       string    `json:"status"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// logFields is what the activity log compares when a claim line changes.
func (cl *ClaimLine) logFields() activitylog.Fields {
	return activitylog.Fields{"status": cl.Status, "notes": activitylog.Value(cl.Notes)}
}
