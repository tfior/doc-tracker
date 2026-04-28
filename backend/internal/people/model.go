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

type Relationship struct {
	PersonID string `json:"person_id"`
	ParentID string `json:"parent_id"`
	CaseID   string `json:"case_id"`
}

type UpdatePersonInput struct {
	FirstName  *string
	LastName   *string
	BirthDate  NullableField
	BirthPlace NullableField
	DeathDate  NullableField
	Notes      NullableField
}

// NullableField carries the three-state value used for optional nullable fields
// in PATCH requests: absent (Set=false), explicit null (Set=true, Valid=false),
// or a string value (Set=true, Valid=true).
type NullableField struct {
	Set   bool
	Valid bool
	Value string
}
