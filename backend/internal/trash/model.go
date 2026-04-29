package trash

import "time"

type TrashedCase struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt time.Time `json:"deleted_at"`
}

type TrashedPerson struct {
	ID        string    `json:"id"`
	CaseID    string    `json:"case_id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt time.Time `json:"deleted_at"`
}

type TrashedLifeEvent struct {
	ID        string    `json:"id"`
	CaseID    string    `json:"case_id"`
	PersonID  string    `json:"person_id"`
	EventType string    `json:"event_type"`
	EventDate *string   `json:"event_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt time.Time `json:"deleted_at"`
}

type TrashedDocument struct {
	ID           string    `json:"id"`
	CaseID       string    `json:"case_id"`
	PersonID     string    `json:"person_id"`
	DocumentType string    `json:"document_type"`
	Title        string    `json:"title"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    time.Time `json:"deleted_at"`
}

type TrashedClaimLine struct {
	ID           string    `json:"id"`
	CaseID       string    `json:"case_id"`
	RootPersonID string    `json:"root_person_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    time.Time `json:"deleted_at"`
}

type CaseTrash struct {
	People     []TrashedPerson     `json:"people"`
	LifeEvents []TrashedLifeEvent  `json:"life_events"`
	Documents  []TrashedDocument   `json:"documents"`
	ClaimLines []TrashedClaimLine  `json:"claim_lines"`
}

type GlobalTrash struct {
	Cases      []TrashedCase      `json:"cases"`
	People     []TrashedPerson    `json:"people"`
	LifeEvents []TrashedLifeEvent `json:"life_events"`
	Documents  []TrashedDocument  `json:"documents"`
	ClaimLines []TrashedClaimLine `json:"claim_lines"`
}
