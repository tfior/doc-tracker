// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"errors"
	"fmt"
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

// InvalidError is invalid input with a message written for the user.
type InvalidError struct {
	Message string
}

func (e *InvalidError) Error() string { return e.Message }

// Invalid returns an InvalidError with the given user-facing message.
func Invalid(format string, args ...any) error {
	return &InvalidError{Message: fmt.Sprintf(format, args...)}
}

// WriteError writes the response for errors that carry a user-facing message
// (ConflictError, InvalidError) and reports whether it did. Handlers call it
// before their own error mapping.
func WriteError(w http.ResponseWriter, err error) bool {
	var ce *ConflictError
	if errors.As(err, &ce) {
		Error(w, http.StatusConflict, "conflict", ce.Message)
		return true
	}
	var ie *InvalidError
	if errors.As(err, &ie) {
		Error(w, http.StatusBadRequest, "invalid_input", ie.Message)
		return true
	}
	return false
}
