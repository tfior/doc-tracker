package cases

import "time"

type Case struct {
	ID                  string    `json:"id"`
	Title               string    `json:"title"`
	Status              string    `json:"status"`
	PrimaryRootPersonID *string   `json:"primary_root_person_id"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ClaimLineSummary struct {
	Total      int `json:"total"`
	Active     int `json:"active"`
	Suspended  int `json:"suspended"`
	Eliminated int `json:"eliminated"`
	Confirmed  int `json:"confirmed"`
}

type DocumentProgress struct {
	NotStarted int `json:"not_started"`
	InProgress int `json:"in_progress"`
	Complete   int `json:"complete"`
}

type CaseDetail struct {
	Case
	ClaimLineSummary ClaimLineSummary `json:"claim_line_summary"`
	DocumentProgress DocumentProgress `json:"document_progress"`
}
