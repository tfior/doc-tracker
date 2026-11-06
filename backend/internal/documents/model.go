// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents

import (
	"time"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/platform"
)

type UpdateDocumentInput struct {
	Version            int     // the version the change is based on; unused on create
	PersonID           *string // update only; on create the person is a separate argument
	LifeEventID        platform.NullString
	Title              *string
	DocumentType       *string
	IssuingAuthority   platform.NullString
	IssueDate          platform.NullString
	RegistrationDate   platform.NullString
	RecordedGivenName  platform.NullString
	RecordedSurname    platform.NullString
	RecordedBirthDate  platform.NullString
	RecordedBirthPlace platform.NullString
	Notes              platform.NullString
	DetailsChecked     *bool
	ForSubmission      *bool

	// Phases, update only. A status must belong to its phase or to "any".
	OfficialCopy PhaseInput
	Amendment    PhaseInput
	Apostille    PhaseInput
	Translation  PhaseInput
}

// PhaseInput is a change to one phase. Since, when given, corrects when the
// phase entered its status; otherwise a change of status sets it to now.
type PhaseInput struct {
	StatusID  *string
	Since     *time.Time
	Reference platform.NullString
}

// checkChange is what a save does to the details check.
type checkChange int

const (
	checkKept    checkChange = iota // left as it is
	checkSet                        // a new check by the saving user, now
	checkCleared                    // cleared, with its time and user
)

// UserRef names a user in a response.
type UserRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PhaseStatus is the resolved status for a single document phase,
// embedded in Document responses.
type PhaseStatus struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Phase          string `json:"phase"`
	ProgressBucket string `json:"progress_bucket"`
}

// DocumentStatus is the full status record returned by GET /api/v1/document-statuses.
type DocumentStatus struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Phase          string `json:"phase"`
	IsSystem       bool   `json:"is_system"`
	ProgressBucket string `json:"progress_bucket"`
}

type Document struct {
	ID                 string     `json:"id"`
	CaseID             string     `json:"case_id"`
	PersonID           string     `json:"person_id"`
	LifeEventID        *string    `json:"life_event_id"`
	DocumentType       string     `json:"document_type"`
	Title              string     `json:"title"`
	IssuingAuthority   *string    `json:"issuing_authority"`
	IssueDate          *string    `json:"issue_date"`
	RegistrationDate   *string    `json:"registration_date"`
	RecordedGivenName  *string    `json:"recorded_given_name"`
	RecordedSurname    *string    `json:"recorded_surname"`
	RecordedBirthDate  *string    `json:"recorded_birth_date"`
	RecordedBirthPlace *string    `json:"recorded_birth_place"`
	DetailsChecked     bool       `json:"details_checked"`
	DetailsCheckedAt   *time.Time `json:"details_checked_at"`
	DetailsCheckedBy   *UserRef   `json:"details_checked_by"`
	ForSubmission      bool       `json:"for_submission"`
	Notes              *string    `json:"notes"`

	OfficialCopyStatus      PhaseStatus `json:"official_copy_status"`
	OfficialCopyStatusSince time.Time   `json:"official_copy_status_since"`
	OfficialCopyReference   *string     `json:"official_copy_reference"`
	AmendmentStatus         PhaseStatus `json:"amendment_status"`
	AmendmentStatusSince    time.Time   `json:"amendment_status_since"`
	AmendmentReference      *string     `json:"amendment_reference"`
	ApostilleStatus         PhaseStatus `json:"apostille_status"`
	ApostilleStatusSince    time.Time   `json:"apostille_status_since"`
	ApostilleReference      *string     `json:"apostille_reference"`
	TranslationStatus       PhaseStatus `json:"translation_status"`
	TranslationStatusSince  time.Time   `json:"translation_status_since"`
	TranslationReference    *string     `json:"translation_reference"`

	// Derived when the document is read; see derive.
	Needed                   bool       `json:"needed"`
	Readiness                *Readiness `json:"readiness"`
	OfficialCopyDaysInStatus int        `json:"official_copy_days_in_status"`
	AmendmentDaysInStatus    int        `json:"amendment_days_in_status"`
	ApostilleDaysInStatus    int        `json:"apostille_days_in_status"`
	TranslationDaysInStatus  int        `json:"translation_days_in_status"`
	CopyAgeDays              *int       `json:"copy_age_days"`

	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// officialCopyKey is the system key of the official copy status, read so
	// that derive can tell whether the document is needed.
	officialCopyKey string
}

// MovedDocument identifies a document whose person changed because its life
// event moved to another person.
type MovedDocument struct {
	ID           string
	Title        string
	FromPersonID string
}

// changesRecordedDetails reports whether a save would change any recorded
// detail of d. Sending a detail's current value is not a change.
func (in *UpdateDocumentInput) changesRecordedDetails(d *Document) bool {
	for _, f := range []struct {
		sent platform.NullString
		was  *string
	}{
		{in.RecordedGivenName, d.RecordedGivenName},
		{in.RecordedSurname, d.RecordedSurname},
		{in.RecordedBirthDate, d.RecordedBirthDate},
		{in.RecordedBirthPlace, d.RecordedBirthPlace},
	} {
		if f.sent.Set && (f.sent.Valid != (f.was != nil) || (f.sent.Valid && f.sent.Value != *f.was)) {
			return true
		}
	}
	return false
}

// checkedByID logs who checked a document's details, so a new check by
// someone else is logged even though the document stays checked.
func checkedByID(u *UserRef) any {
	if u == nil {
		return nil
	}
	return u.ID
}

// logFields is what the activity log compares when a document changes. Phase
// statuses are logged by label, which is what a reader needs.
func (d *Document) logFields() activitylog.Fields {
	since := func(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
	return activitylog.Fields{
		"person_id":            d.PersonID,
		"life_event_id":        activitylog.Value(d.LifeEventID),
		"document_type":        d.DocumentType,
		"title":                d.Title,
		"issuing_authority":    activitylog.Value(d.IssuingAuthority),
		"issue_date":           activitylog.Value(d.IssueDate),
		"registration_date":    activitylog.Value(d.RegistrationDate),
		"recorded_given_name":  activitylog.Value(d.RecordedGivenName),
		"recorded_surname":     activitylog.Value(d.RecordedSurname),
		"recorded_birth_date":  activitylog.Value(d.RecordedBirthDate),
		"recorded_birth_place": activitylog.Value(d.RecordedBirthPlace),
		"notes":                activitylog.Value(d.Notes),
		"details_checked":      d.DetailsChecked,
		"details_checked_by":   checkedByID(d.DetailsCheckedBy),
		"for_submission":       d.ForSubmission,

		"official_copy_status":       d.OfficialCopyStatus.Label,
		"official_copy_status_since": since(d.OfficialCopyStatusSince),
		"official_copy_reference":    activitylog.Value(d.OfficialCopyReference),
		"amendment_status":           d.AmendmentStatus.Label,
		"amendment_status_since":     since(d.AmendmentStatusSince),
		"amendment_reference":        activitylog.Value(d.AmendmentReference),
		"apostille_status":           d.ApostilleStatus.Label,
		"apostille_status_since":     since(d.ApostilleStatusSince),
		"apostille_reference":        activitylog.Value(d.ApostilleReference),
		"translation_status":         d.TranslationStatus.Label,
		"translation_status_since":   since(d.TranslationStatusSince),
		"translation_reference":      activitylog.Value(d.TranslationReference),
	}
}
