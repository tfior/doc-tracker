// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsID reports whether s is a well-formed ID.
func IsID(s string) bool { return uuidPattern.MatchString(s) }

// Decode reads a JSON request body into dst. Unknown fields are refused, so a
// misspelt field is reported instead of silently ignored.
func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) {
		return Invalid("The request has no body.")
	}
	if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		field = strings.Trim(field, `"`)
		return InvalidField(field, "unknown_field", fmt.Sprintf("%q is not a recognised field.", field))
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		return InvalidField(typeErr.Field, "invalid_value", "This value has the wrong type.")
	}
	return Invalid("The request body is not valid JSON.")
}

// Validator collects every problem with a request, so they are reported
// together instead of one at a time.
type Validator struct {
	errs []FieldError
}

// Add records a problem with a field.
func (v *Validator) Add(field, code, message string) {
	v.errs = append(v.errs, FieldError{Field: field, Code: code, Message: message})
}

// Required records a problem if value is blank.
func (v *Validator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.Add(field, "required", "This field is required.")
	}
}

// NotBlank records a problem if a supplied value is blank. A value that was
// not supplied (nil) is fine.
func (v *Validator) NotBlank(field string, value *string) {
	if value != nil && strings.TrimSpace(*value) == "" {
		v.Add(field, "required", "This field cannot be empty.")
	}
}

// Version records a problem if the record version was not supplied.
func (v *Validator) Version(version *int) {
	if version == nil {
		v.Add("version", "required", "The version of the record being changed is required.")
	}
}

// ID records a problem if a supplied value is not a well-formed ID.
func (v *Validator) ID(field, value string) {
	if value != "" && !IsID(value) {
		v.Add(field, "invalid_id", "This is not a valid ID.")
	}
}

// IDs records a problem for each list entry that is not a well-formed ID.
func (v *Validator) IDs(field string, values []string) {
	for _, id := range values {
		if !IsID(id) {
			v.Add(field, "invalid_id", fmt.Sprintf("%q is not a valid ID.", id))
		}
	}
}

// OneOf records a problem if a supplied value is not one of allowed.
func (v *Validator) OneOf(field, value string, allowed map[string]bool) {
	if value != "" && !allowed[value] {
		v.Add(field, "invalid_value", fmt.Sprintf("%q is not an accepted value.", value))
	}
}

// Date records a problem if a supplied, non-null date is not a real date in
// the form YYYY-MM-DD. Other forms are refused rather than interpreted, so a
// date is never read the wrong way round.
func (v *Validator) Date(field string, n NullString) {
	if !n.Set || !n.Valid {
		return
	}
	if _, err := time.Parse("2006-01-02", n.Value); err != nil {
		v.Add(field, "invalid_date", "Enter a real date in the form YYYY-MM-DD, for example 1890-03-15.")
	}
}

// Err returns the collected problems as an InvalidError, or nil if there are none.
func (v *Validator) Err() error {
	if len(v.errs) == 0 {
		return nil
	}
	return &InvalidError{Message: "Some fields are invalid.", Fields: v.errs}
}

// Trim trims a supplied string in place.
func Trim(s *string) {
	if s != nil {
		*s = strings.TrimSpace(*s)
	}
}

// QueryVersion reads the required ?version= parameter of a delete.
func QueryVersion(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("version")
	if raw == "" {
		return 0, InvalidField("version", "required", "The version of the record being deleted is required.")
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return 0, InvalidField("version", "invalid_value", "The version must be a positive whole number.")
	}
	return v, nil
}

// ValidIDs refuses requests whose URL carries a malformed ID. Under
// /api/v1/cases/, every second path segment, starting with the case's, is an ID.
func ValidIDs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rest, ok := strings.CutPrefix(r.URL.Path, "/api/v1/cases/"); ok {
			for i, seg := range strings.Split(rest, "/") {
				if i%2 == 0 && seg != "" && !IsID(seg) {
					WriteError(w, InvalidField("id", "invalid_id", fmt.Sprintf("%q is not a valid ID.", seg)))
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Items is the envelope for lists that are returned in full.
type Items struct {
	Items any `json:"items"`
	Total int `json:"total"`
}

// ParseTimestamp reads a timestamp sent by a client, in RFC 3339. One sent
// without a time zone offset is read as UTC.
func ParseTimestamp(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}
