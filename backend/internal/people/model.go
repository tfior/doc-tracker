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
	ID        string `json:"id"`
	CaseID    string `json:"case_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	// Birth and death are read from the person's birth and death life events.
	BirthDate  *platform.QualifiedDate `json:"birth_date"`
	BirthPlace *string                 `json:"birth_place"`
	DeathDate  *platform.QualifiedDate `json:"death_date"`
	DeathPlace *string                 `json:"death_place"`
	Notes      *string                 `json:"notes"`
	ParentIDs  []string                `json:"parent_ids"`
	Version    int                     `json:"version"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
}

type Relationship struct {
	PersonID string `json:"person_id"`
	ParentID string `json:"parent_id"`
	CaseID   string `json:"case_id"`
}

type UpdatePersonInput struct {
	Version   int // the version the change is based on; unused on create
	FirstName *string
	LastName  *string
	// Birth and death are written to the person's birth and death life events.
	BirthDate  platform.NullQualifiedDate
	BirthPlace platform.NullString
	DeathDate  platform.NullQualifiedDate
	DeathPlace platform.NullString
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

// logFields is what the activity log compares when a person changes. Birth
// and death are logged against their life events, not here.
func (p *Person) logFields() activitylog.Fields {
	return activitylog.Fields{
		"first_name": p.FirstName,
		"last_name":  p.LastName,
		"notes":      activitylog.Value(p.Notes),
	}
}
