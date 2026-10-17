// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package people

import (
	"time"

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
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Relationship struct {
	PersonID string `json:"person_id"`
	ParentID string `json:"parent_id"`
	CaseID   string `json:"case_id"`
}

type UpdatePersonInput struct {
	FirstName  *string
	LastName   *string
	BirthDate  platform.NullString
	BirthPlace platform.NullString
	DeathDate  platform.NullString
	Notes      platform.NullString
}
