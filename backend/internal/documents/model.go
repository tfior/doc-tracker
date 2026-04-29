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
	Title               *string
	DocumentType        *string
	LifeEventID         NullableField // used on create only; ignored by update handler
	IssuingAuthority    NullableField
	IssueDate           NullableField
	RecordedDate        NullableField
	RecordedGivenName   NullableField
	RecordedSurname     NullableField
	RecordedBirthDate   NullableField
	RecordedBirthPlace  NullableField
	Notes               NullableField
	IsVerified          *bool
}

type ReassignDocumentInput struct {
	PersonID    string
	LifeEventID NullableField
}

type Document struct {
	ID                  string     `json:"id"`
	CaseID              string     `json:"case_id"`
	PersonID            string     `json:"person_id"`
	LifeEventID         *string    `json:"life_event_id"`
	DocumentType        string     `json:"document_type"`
	Title               string     `json:"title"`
	IssuingAuthority    *string    `json:"issuing_authority"`
	IssueDate           *string    `json:"issue_date"`
	RecordedDate        *string    `json:"recorded_date"`
	RecordedGivenName   *string    `json:"recorded_given_name"`
	RecordedSurname     *string    `json:"recorded_surname"`
	RecordedBirthDate   *string    `json:"recorded_birth_date"`
	RecordedBirthPlace  *string    `json:"recorded_birth_place"`
	IsVerified          bool       `json:"is_verified"`
	VerifiedAt          *time.Time `json:"verified_at"`
	Notes               *string    `json:"notes"`
	Status              string     `json:"status"`
	StatusKey           *string    `json:"status_key"`
	ProgressBucket      string     `json:"progress_bucket"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
