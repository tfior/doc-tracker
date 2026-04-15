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
