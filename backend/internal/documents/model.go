package documents

import "time"

// NullableField carries the three-state value for optional nullable fields in
// PATCH requests: absent (Set=false), explicit null (Set=true, Valid=false),
// or a value (Set=true, Valid=true).
type NullableField struct {
	Set   bool
	Valid bool
	Value string
}

type UpdateDocumentInput struct {
	Title              *string
	DocumentType       *string
	LifeEventID        NullableField // used on create only; ignored by update handler
	IssuingAuthority   NullableField
	IssueDate          NullableField
	RecordedDate       NullableField
	RecordedGivenName  NullableField
	RecordedSurname    NullableField
	RecordedBirthDate  NullableField
	RecordedBirthPlace NullableField
	Notes              NullableField
	IsVerified         *bool
}

type ReassignDocumentInput struct {
	PersonID    string
	LifeEventID NullableField
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
