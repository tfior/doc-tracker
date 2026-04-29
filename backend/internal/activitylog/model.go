package activitylog

// FieldChange records the new value of a single field in an update operation.
// "From" values are omitted in MVP; they can be added when the activity feed
// UI is built by comparing consecutive log entries.
type FieldChange struct {
	Field string `json:"field"`
	To    any    `json:"to"`
}

type InsertParams struct {
	CaseID     string
	UserID     string
	Action     string // created | updated | deleted
	EntityType string // case | person | person_relationship | life_event | document | file_attachment | claim_line
	EntityID   string
	EntityName string
	Changes    []FieldChange // nil for create/delete; populated for updates
}
