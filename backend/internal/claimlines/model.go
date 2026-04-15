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
