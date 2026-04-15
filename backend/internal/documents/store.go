package documents

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error)
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM documents WHERE case_id = $1`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count documents: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			d.id::text, d.case_id::text, d.person_id::text, d.life_event_id::text,
			d.document_type::text, d.title,
			d.issuing_authority, d.issue_date::text,
			d.recorded_date::text, d.recorded_given_name, d.recorded_surname,
			d.recorded_birth_date::text, d.recorded_birth_place,
			d.is_verified, d.verified_at, d.notes,
			ds.label, ds.system_key::text, ds.progress_bucket::text,
			d.created_at, d.updated_at
		FROM documents d
		JOIN document_statuses ds ON ds.id = d.status_id
		WHERE d.case_id = $1
		ORDER BY d.created_at ASC
		LIMIT $2 OFFSET $3`,
		caseID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query documents: %w", err)
	}
	defer rows.Close()

	items := []Document{}
	for rows.Next() {
		var d Document
		var lifeEventID, issuingAuthority, issueDate sql.NullString
		var recordedDate, recordedGivenName, recordedSurname sql.NullString
		var recordedBirthDate, recordedBirthPlace, notes sql.NullString
		var statusKey sql.NullString
		var verifiedAt sql.NullTime

		if err := rows.Scan(
			&d.ID, &d.CaseID, &d.PersonID, &lifeEventID,
			&d.DocumentType, &d.Title,
			&issuingAuthority, &issueDate,
			&recordedDate, &recordedGivenName, &recordedSurname,
			&recordedBirthDate, &recordedBirthPlace,
			&d.IsVerified, &verifiedAt, &notes,
			&d.Status, &statusKey, &d.ProgressBucket,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan document: %w", err)
		}

		setString(&d.LifeEventID, lifeEventID)
		setString(&d.IssuingAuthority, issuingAuthority)
		setString(&d.IssueDate, issueDate)
		setString(&d.RecordedDate, recordedDate)
		setString(&d.RecordedGivenName, recordedGivenName)
		setString(&d.RecordedSurname, recordedSurname)
		setString(&d.RecordedBirthDate, recordedBirthDate)
		setString(&d.RecordedBirthPlace, recordedBirthPlace)
		setString(&d.Notes, notes)
		setString(&d.StatusKey, statusKey)

		if verifiedAt.Valid {
			t := verifiedAt.Time.UTC()
			d.VerifiedAt = &t
		}

		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate documents: %w", err)
	}

	return items, total, nil
}

func setString(dest **string, ns sql.NullString) {
	if ns.Valid {
		s := ns.String
		*dest = &s
	}
}
