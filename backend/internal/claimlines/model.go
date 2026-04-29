package claimlines

import "time"

type ClaimLine struct {
	ID           string    `json:"id"`
	CaseID       string    `json:"case_id"`
	RootPersonID string    `json:"root_person_id"`
	Status       string    `json:"status"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NullableField carries the three-state value for optional nullable fields in
// PATCH requests: absent (Set=false), explicit null (Set=true, Valid=false),
// or a value (Set=true, Valid=true).
type NullableField struct {
	Set   bool
	Valid bool
	Value string
}
