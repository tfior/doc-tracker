// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents

import (
	"time"

	"github.com/tfior/doc-tracker/platform"
)

type UpdateDocumentInput struct {
	Title              *string
	DocumentType       *string
	LifeEventID        platform.NullString // used on create only; ignored by update handler
	IssuingAuthority   platform.NullString
	IssueDate          platform.NullString
	RecordedDate       platform.NullString
	RecordedGivenName  platform.NullString
	RecordedSurname    platform.NullString
	RecordedBirthDate  platform.NullString
	RecordedBirthPlace platform.NullString
	Notes              platform.NullString
	IsVerified         *bool
}

type ReassignDocumentInput struct {
	PersonID    string
	LifeEventID platform.NullString
}

type TransitionStatusInput struct {
	Phase    string
	StatusID string
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
	ID                 string      `json:"id"`
	CaseID             string      `json:"case_id"`
	PersonID           string      `json:"person_id"`
	LifeEventID        *string     `json:"life_event_id"`
	DocumentType       string      `json:"document_type"`
	Title              string      `json:"title"`
	IssuingAuthority   *string     `json:"issuing_authority"`
	IssueDate          *string     `json:"issue_date"`
	RecordedDate       *string     `json:"recorded_date"`
	RecordedGivenName  *string     `json:"recorded_given_name"`
	RecordedSurname    *string     `json:"recorded_surname"`
	RecordedBirthDate  *string     `json:"recorded_birth_date"`
	RecordedBirthPlace *string     `json:"recorded_birth_place"`
	IsVerified         bool        `json:"is_verified"`
	VerifiedAt         *time.Time  `json:"verified_at"`
	Notes              *string     `json:"notes"`
	OfficialCopyStatus PhaseStatus `json:"official_copy_status"`
	AmendmentStatus    PhaseStatus `json:"amendment_status"`
	ApostilleStatus    PhaseStatus `json:"apostille_status"`
	TranslationStatus  PhaseStatus `json:"translation_status"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// MovedDocument identifies a document whose person changed because its life
// event moved to another person.
type MovedDocument struct {
	ID    string
	Title string
}
