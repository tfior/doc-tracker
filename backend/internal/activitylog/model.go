// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package activitylog

import (
	"reflect"
	"sort"
)

// FieldChange records one field's old and new value in an update. A nil From
// or To means null.
type FieldChange struct {
	Field string `json:"field"`
	From  any    `json:"from"`
	To    any    `json:"to"`
}

// Fields is a record's loggable fields, by field name. Each module builds one
// for its entity, before and after a change, and Diff compares them.
type Fields map[string]any

// Diff returns a change for every field whose value differs between before
// and after, in field-name order. Fields equal in both are left out.
func Diff(before, after Fields) []FieldChange {
	names := make([]string, 0, len(after))
	for name := range after {
		names = append(names, name)
	}
	sort.Strings(names)
	var changes []FieldChange
	for _, name := range names {
		if !reflect.DeepEqual(before[name], after[name]) {
			changes = append(changes, FieldChange{Field: name, From: before[name], To: after[name]})
		}
	}
	return changes
}

// Value turns an optional string into a loggable value: nil for null.
func Value(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

type InsertParams struct {
	CaseID     string
	UserID     string
	Action     string // created | updated | deleted | restored | permanently_deleted
	EntityType string // case | person | person_relationship | life_event | document | file_attachment | claim_line
	EntityID   string
	EntityName string
	Changes    []FieldChange // only for updates; an update with no changes is not logged
}
