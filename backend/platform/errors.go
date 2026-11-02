// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// ErrNotFound is the sentinel every module's not-found error wraps, so that one
// module can recognise "not found" from another without importing it.
var ErrNotFound = errors.New("not found")

// MapNotFound returns target when err is any module's not-found error, and err
// unchanged otherwise. A module uses it to report another module's missing
// entity as its own.
func MapNotFound(err, target error) error {
	if errors.Is(err, ErrNotFound) {
		return target
	}
	return err
}

// ErrConflict is the sentinel every conflict error wraps: a write refused
// because of the current state of the data, not because the input is invalid.
var ErrConflict = errors.New("conflict")

// ConflictError is a conflict with a message written for the user, saying what
// is wrong and what to do about it.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string { return e.Message }

func (e *ConflictError) Is(target error) bool { return target == ErrConflict }

// Conflict returns a ConflictError with the given user-facing message.
func Conflict(format string, args ...any) error {
	return &ConflictError{Message: fmt.Sprintf(format, args...)}
}

// FieldError is one field at fault in an invalid request.
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// InvalidError is invalid input with a message written for the user, and the
// fields at fault when the problem can be tied to them.
type InvalidError struct {
	Message string
	Fields  []FieldError
}

func (e *InvalidError) Error() string { return e.Message }

// Invalid returns an InvalidError with the given user-facing message.
func Invalid(format string, args ...any) error {
	return &InvalidError{Message: fmt.Sprintf(format, args...)}
}

// InvalidField returns an InvalidError for a single field.
func InvalidField(field, code, message string) error {
	return &InvalidError{Message: message, Fields: []FieldError{{Field: field, Code: code, Message: message}}}
}

// ErrVersionConflict means the record changed since the client read it.
var ErrVersionConflict = errors.New("version conflict")

// VersionConflict returns the error for a write based on an out-of-date version.
func VersionConflict() error { return ErrVersionConflict }

// CheckVersion returns ErrVersionConflict unless the client's version is the
// record's current one.
func CheckVersion(current, sent int) error {
	if current != sent {
		return ErrVersionConflict
	}
	return nil
}

// WriteError writes the response for errors that carry their own meaning for
// the user (version conflicts, conflicts, invalid input) and reports whether
// it did. Handlers call it before their own error mapping.
func WriteError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, ErrVersionConflict) {
		Error(w, http.StatusConflict, "version_conflict",
			"This record was changed by someone else since you opened it. Reload it and make your change again.")
		return true
	}
	var ce *ConflictError
	if errors.As(err, &ce) {
		Error(w, http.StatusConflict, "conflict", ce.Message)
		return true
	}
	var ie *InvalidError
	if errors.As(err, &ie) {
		body := map[string]any{"code": "invalid_input", "message": ie.Message}
		if len(ie.Fields) > 0 {
			body["fields"] = ie.Fields
		}
		JSON(w, http.StatusBadRequest, map[string]any{"error": body})
		return true
	}
	return false
}

// ServerError logs an unexpected error with its cause and writes a 500. The
// cause is never sent to the client.
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	Error(w, http.StatusInternalServerError, "internal", "Something went wrong on the server. Please try again.")
}
