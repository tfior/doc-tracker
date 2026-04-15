package people

import "time"

type Person struct {
	ID         string    `json:"id"`
	CaseID     string    `json:"case_id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	BirthDate  *string   `json:"birth_date"`
	BirthPlace *string   `json:"birth_place"`
	DeathDate  *string   `json:"death_date"`
	Notes      *string   `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
