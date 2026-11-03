// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package lifeevents

import (
	"time"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/platform"
)

type LifeEvent struct {
	ID               string                  `json:"id"`
	CaseID           string                  `json:"case_id"`
	PersonID         string                  `json:"person_id"`
	EventType        string                  `json:"event_type"`
	EventDate        *platform.QualifiedDate `json:"event_date"`
	EventPlace       *string                 `json:"event_place"`
	SpouseName       *string                 `json:"spouse_name"`
	SpouseBirthDate  *platform.QualifiedDate `json:"spouse_birth_date"`
	SpouseBirthPlace *string                 `json:"spouse_birth_place"`
	Notes            *string                 `json:"notes"`
	HasDocuments     bool                    `json:"has_documents"`
	Version          int                     `json:"version"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

type UpdateLifeEventInput struct {
	Version          int     // the version the change is based on; unused on create
	PersonID         *string // moves the event, and its documents, to another person
	EventType        *string
	EventDate        platform.NullQualifiedDate
	EventPlace       platform.NullString
	SpouseName       platform.NullString
	SpouseBirthDate  platform.NullQualifiedDate
	SpouseBirthPlace platform.NullString
	Notes            platform.NullString
}

// logFields is what the activity log compares when a life event changes.
func (le *LifeEvent) logFields() activitylog.Fields {
	return activitylog.Fields{
		"person_id":          le.PersonID,
		"event_type":         le.EventType,
		"event_date":         dateValue(le.EventDate),
		"event_place":        activitylog.Value(le.EventPlace),
		"spouse_name":        activitylog.Value(le.SpouseName),
		"spouse_birth_date":  dateValue(le.SpouseBirthDate),
		"spouse_birth_place": activitylog.Value(le.SpouseBirthPlace),
		"notes":              activitylog.Value(le.Notes),
	}
}

// dateValue turns an optional qualified date into a loggable value: nil for
// null.
func dateValue(d *platform.QualifiedDate) any {
	if d == nil {
		return nil
	}
	return *d
}

// DeletionImpact is what deleting a life event would take with it.
type DeletionImpact struct {
	Documents int `json:"documents"`
}
