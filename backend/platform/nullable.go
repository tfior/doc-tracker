// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"database/sql"
	"encoding/json"
)

// NullString is a JSON-aware nullable string that distinguishes between three
// states: field absent from JSON (Set=false), field explicitly null (Set=true,
// Valid=false), and field with a value (Set=true, Valid=true).
//
// Use it in PATCH request structs for optional nullable fields so that
// "omit = no change" and "null = clear" can be handled correctly.
type NullString struct {
	Set   bool
	Valid bool
	Value string
}

func (n *NullString) UnmarshalJSON(b []byte) error {
	n.Set = true
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(b, &n.Value)
}

// SQLValue returns a sql.NullString suitable for use as a query parameter.
func (n NullString) SQLValue() sql.NullString {
	return sql.NullString{String: n.Value, Valid: n.Valid}
}
