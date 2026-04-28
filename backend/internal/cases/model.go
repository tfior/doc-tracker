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
	Total            int `json:"total"`
	NotYetResearched int `json:"not_yet_researched"`
	Researching      int `json:"researching"`
	Paused           int `json:"paused"`
	Ineligible       int `json:"ineligible"`
	Eligible         int `json:"eligible"`
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
