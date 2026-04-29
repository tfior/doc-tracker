package lifeevents

import "time"

type LifeEvent struct {
	ID               string    `json:"id"`
	CaseID           string    `json:"case_id"`
	PersonID         string    `json:"person_id"`
	EventType        string    `json:"event_type"`
	EventDate        *string   `json:"event_date"`
	EventPlace       *string   `json:"event_place"`
	SpouseName       *string   `json:"spouse_name"`
	SpouseBirthDate  *string   `json:"spouse_birth_date"`
	SpouseBirthPlace *string   `json:"spouse_birth_place"`
	Notes            *string   `json:"notes"`
	HasDocuments     bool      `json:"has_documents"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UpdateLifeEventInput struct {
	EventType        *string
	EventDate        NullableField
	EventPlace       NullableField
	SpouseName       NullableField
	SpouseBirthDate  NullableField
	SpouseBirthPlace NullableField
	Notes            NullableField
}

// NullableField carries the three-state value for optional nullable fields in
// PATCH requests: absent (Set=false), explicit null (Set=true, Valid=false),
// or a value (Set=true, Valid=true).
type NullableField struct {
	Set   bool
	Valid bool
	Value string
}
