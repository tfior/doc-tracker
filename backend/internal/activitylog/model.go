// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package activitylog

import "github.com/tfior/doc-tracker/platform"

// FieldChange records the new value of a single field in an update operation.
// "From" values are omitted in MVP; they can be added when the activity feed
// UI is built by comparing consecutive log entries.
type FieldChange struct {
	Field string `json:"field"`
	To    any    `json:"to"`
}

type InsertParams struct {
	CaseID     string
	UserID     string
	Action     string // created | updated | deleted
	EntityType string // case | person | person_relationship | life_event | document | file_attachment | claim_line
	EntityID   string
	EntityName string
	Changes    []FieldChange // nil for create/delete; populated for updates
}

// AddString appends a change for a non-nullable field when it was supplied.
func AddString(changes []FieldChange, field string, v *string) []FieldChange {
	if v == nil {
		return changes
	}
	return append(changes, FieldChange{Field: field, To: *v})
}

// AddBool appends a change for a boolean field when it was supplied.
func AddBool(changes []FieldChange, field string, v *bool) []FieldChange {
	if v == nil {
		return changes
	}
	return append(changes, FieldChange{Field: field, To: *v})
}

// AddNullable appends a change for a nullable field when it was supplied,
// recording null when the field was cleared.
func AddNullable(changes []FieldChange, field string, n platform.NullString) []FieldChange {
	if !n.Set {
		return changes
	}
	var to any
	if n.Valid {
		to = n.Value
	}
	return append(changes, FieldChange{Field: field, To: to})
}
