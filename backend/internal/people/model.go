// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package people

import (
	"time"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/platform"
)

type Person struct {
	ID         string    `json:"id"`
	CaseID     string    `json:"case_id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	BirthDate  *string   `json:"birth_date"`
	BirthPlace *string   `json:"birth_place"`
	DeathDate  *string   `json:"death_date"`
	Notes      *string   `json:"notes"`
	ParentIDs  []string  `json:"parent_ids"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Relationship struct {
	PersonID string `json:"person_id"`
	ParentID string `json:"parent_id"`
	CaseID   string `json:"case_id"`
}

type UpdatePersonInput struct {
	Version    int // the version the change is based on; unused on create
	FirstName  *string
	LastName   *string
	BirthDate  platform.NullString
	BirthPlace platform.NullString
	DeathDate  platform.NullString
	Notes      platform.NullString
	// ParentIDs and ChildIDs, when supplied, are the person's complete lists of
	// active parents and children; links not listed are removed.
	ParentIDs *[]string
	ChildIDs  *[]string
}

// DeletionImpact is what deleting a person would take with them.
type DeletionImpact struct {
	LifeEvents int `json:"life_events"`
	Documents  int `json:"documents"`
	ClaimLines int `json:"claim_lines"`
}

// logFields is what the activity log compares when a person changes.
func (p *Person) logFields() activitylog.Fields {
	return activitylog.Fields{
		"first_name":  p.FirstName,
		"last_name":   p.LastName,
		"birth_date":  activitylog.Value(p.BirthDate),
		"birth_place": activitylog.Value(p.BirthPlace),
		"death_date":  activitylog.Value(p.DeathDate),
		"notes":       activitylog.Value(p.Notes),
	}
}
