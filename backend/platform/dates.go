// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Dates that describe people, events and documents may be known only in part,
// so they are text, never a calendar date with invented parts. See
// domain-model.md (Field Types, Dates) and api-conventions.md (Dates).

var partialDatePattern = regexp.MustCompile(`^(\d{4})(?:-(\d{2})(?:-(\d{2}))?)?$`)

// PartialDate is a date known to a year, a year and month, or a full date.
// Month and Day are 0 when not known.
type PartialDate struct {
	Year, Month, Day int
}

// ParsePartialDate reads "1890", "1890-03" or "1890-03-15". Any other form,
// and any date that does not exist, is refused rather than interpreted.
func ParsePartialDate(s string) (PartialDate, bool) {
	m := partialDatePattern.FindStringSubmatch(s)
	if m == nil {
		return PartialDate{}, false
	}
	var d PartialDate
	d.Year, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		d.Month, _ = strconv.Atoi(m[2])
	}
	if m[3] != "" {
		d.Day, _ = strconv.Atoi(m[3])
	}
	if d.Year < 1 || (m[2] != "" && (d.Month < 1 || d.Month > 12)) ||
		(m[3] != "" && (d.Day < 1 || d.Day > daysIn(d.Year, d.Month))) {
		return PartialDate{}, false
	}
	return d, true
}

func daysIn(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// Earliest is the first day the date covers: 1 January 1890 for "1890".
func (d PartialDate) Earliest() time.Time {
	month, day := max(d.Month, 1), max(d.Day, 1)
	return time.Date(d.Year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// Latest is the last day the date covers: 31 December 1890 for "1890".
func (d PartialDate) Latest() time.Time {
	month, day := d.Month, d.Day
	if month == 0 {
		month = 12
	}
	if day == 0 {
		day = daysIn(d.Year, month)
	}
	return time.Date(d.Year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func (d PartialDate) String() string {
	switch {
	case d.Month == 0:
		return fmt.Sprintf("%04d", d.Year)
	case d.Day == 0:
		return fmt.Sprintf("%04d-%02d", d.Year, d.Month)
	default:
		return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
	}
}

var qualifiers = map[string]bool{"exact": true, "about": true, "before": true, "after": true, "between": true}

// QualifiedDate is a partial date with a qualifier saying how it is known,
// and for "between" only, the end of the range.
type QualifiedDate struct {
	Value     string  `json:"value"`
	Qualifier string  `json:"qualifier"`
	End       *string `json:"end,omitempty"`
}

// Earliest is the first day the date allows, or false when it is open, as
// for "before 1915". "about" is not widened: any margin would be invented.
func (q QualifiedDate) Earliest() (time.Time, bool) {
	v, ok := ParsePartialDate(q.Value)
	if !ok {
		return time.Time{}, false
	}
	switch q.Qualifier {
	case "before":
		return time.Time{}, false
	case "after":
		return v.Latest().AddDate(0, 0, 1), true
	default:
		return v.Earliest(), true
	}
}

// Latest is the last day the date allows, or false when it is open, as for
// "after 1888".
func (q QualifiedDate) Latest() (time.Time, bool) {
	v, ok := ParsePartialDate(q.Value)
	if !ok {
		return time.Time{}, false
	}
	switch q.Qualifier {
	case "before":
		return v.Earliest().AddDate(0, 0, -1), true
	case "after":
		return time.Time{}, false
	case "between":
		if q.End == nil {
			return time.Time{}, false
		}
		end, ok := ParsePartialDate(*q.End)
		if !ok {
			return time.Time{}, false
		}
		return end.Latest(), true
	default:
		return v.Latest(), true
	}
}

// ScanQualifiedDate builds a qualified date from the three columns it is
// stored in: the date, its qualifier, and the end of a range.
func ScanQualifiedDate(value, qualifier, end sql.NullString) *QualifiedDate {
	if !value.Valid {
		return nil
	}
	d := QualifiedDate{Value: value.String, Qualifier: qualifier.String}
	if end.Valid {
		d.End = &end.String
	}
	return &d
}

// NullQualifiedDate is a qualified date in a request body, with the same three
// states as NullString: absent, null, or a value. The value is kept as sent
// until Validator.QualifiedDate checks it and fills in Date.
type NullQualifiedDate struct {
	Set   bool
	Valid bool
	Date  QualifiedDate
	raw   json.RawMessage
}

func (n *NullQualifiedDate) UnmarshalJSON(b []byte) error {
	n.Set = true
	n.Valid = string(b) != "null"
	n.raw = append(json.RawMessage(nil), b...)
	return nil
}

const partialDateMessage = "Enter a year, a year and month, or a full date, for example 1890, 1890-03 or 1890-03-15."

// PartialDate records a problem if a supplied, non-null date is not a
// partial date that exists.
func (v *Validator) PartialDate(field string, n NullString) {
	if !n.Set || !n.Valid {
		return
	}
	if _, ok := ParsePartialDate(n.Value); !ok {
		v.Add(field, "invalid_date", partialDateMessage)
	}
}

// QualifiedDate checks a supplied, non-null qualified date, recording every
// problem against the part at fault ("event_date.end"). When there are none
// it fills in n.Date.
func (v *Validator) QualifiedDate(field string, n *NullQualifiedDate) {
	if !n.Set || !n.Valid {
		return
	}
	var parts map[string]json.RawMessage
	if bytes.HasPrefix(bytes.TrimSpace(n.raw), []byte("{")) {
		_ = json.Unmarshal(n.raw, &parts)
	}
	if parts == nil {
		v.Add(field, "invalid_value", "Enter a date with a qualifier, such as exact or about.")
		return
	}
	before := len(v.errs)

	// text reads a part. A part left out, null, or blank is missing; one of
	// the wrong type is reported here and not checked further.
	wrongType := map[string]bool{}
	text := func(part string) (string, bool) {
		raw, ok := parts[part]
		if !ok || string(raw) == "null" {
			return "", false
		}
		var s string
		if json.Unmarshal(raw, &s) != nil {
			v.Add(field+"."+part, "invalid_value", "This value has the wrong type.")
			wrongType[part] = true
			return "", false
		}
		return s, s != ""
	}
	for part := range parts {
		if part != "value" && part != "qualifier" && part != "end" {
			v.Add(field+"."+part, "unknown_field", fmt.Sprintf("%q is not a recognised field.", part))
		}
	}

	var q QualifiedDate
	value, hasValue := text("value")
	start, startOK := ParsePartialDate(value)
	switch {
	case wrongType["value"]:
	case !hasValue:
		v.Add(field+".value", "required", "Enter the date.")
	case !startOK:
		v.Add(field+".value", "invalid_date", partialDateMessage)
	}
	q.Value = value

	qualifier, hasQualifier := text("qualifier")
	switch {
	case wrongType["qualifier"]:
	case !hasQualifier:
		v.Add(field+".qualifier", "required", "Choose how well the date is known.")
	case !qualifiers[qualifier]:
		v.Add(field+".qualifier", "invalid_value", fmt.Sprintf("%q is not an accepted value.", qualifier))
	}
	q.Qualifier = qualifier

	endText, hasEnd := text("end")
	end, endOK := ParsePartialDate(endText)
	switch {
	case wrongType["end"]:
	case qualifier == "between" && !hasEnd:
		v.Add(field+".end", "required", "Enter the end of the range.")
	case qualifier != "between" && hasEnd && qualifiers[qualifier]:
		v.Add(field+".end", "invalid_value", "Only a date range has an end date.")
	case hasEnd && !endOK:
		v.Add(field+".end", "invalid_date", partialDateMessage)
	case qualifier == "between" && startOK && !end.Earliest().After(start.Latest()):
		v.Add(field+".end", "invalid_value", "The end of the range must come after its start.")
	}
	if hasEnd {
		q.End = &endText
	}

	if len(v.errs) == before {
		n.Date = q
	}
}
