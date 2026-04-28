package documents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("document not found")

var validStatusKeys = map[string]bool{
	"pending": true, "collected": true, "verified": true, "unobtainable": true,
}

type Store interface {
	ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error)
	CreateDocument(ctx context.Context, caseID, personID, docType, title string, input UpdateDocumentInput) (*Document, error)
	UpdateDocument(ctx context.Context, caseID, docID string, input UpdateDocumentInput) (*Document, error)
	DeleteDocument(ctx context.Context, caseID, docID string) error
	TransitionStatus(ctx context.Context, caseID, docID, statusKey string) (*Document, error)
	ReassignDocument(ctx context.Context, caseID, docID string, input ReassignDocumentInput) (*Document, error)
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
		`SELECT COUNT(*) FROM documents WHERE case_id = $1 AND deleted_at IS NULL`, caseID,
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
		WHERE d.case_id = $1 AND d.deleted_at IS NULL
		ORDER BY d.created_at ASC
		LIMIT $2 OFFSET $3`,
		caseID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query documents: %w", err)
	}
	defer rows.Close()

	items := []Document{}
	for rows.Next() {
		d, err := scanDocument(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scan document: %w", err)
		}
		items = append(items, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate documents: %w", err)
	}

	return items, total, nil
}

func (s *store) CreateDocument(ctx context.Context, caseID, personID, docType, title string, input UpdateDocumentInput) (*Document, error) {
	if err := s.requirePersonInCase(ctx, caseID, personID); err != nil {
		return nil, err
	}
	if input.LifeEventID.Set && input.LifeEventID.Valid {
		if err := s.requireLifeEventInCase(ctx, caseID, input.LifeEventID.Value); err != nil {
			return nil, err
		}
	}

	var statusID string
	if err := s.db.QueryRowContext(ctx,
		`SELECT id::text FROM document_statuses WHERE system_key::text = 'pending'`,
	).Scan(&statusID); err != nil {
		return nil, fmt.Errorf("get pending status: %w", err)
	}

	var lifeEventID sql.NullString
	if input.LifeEventID.Set && input.LifeEventID.Valid {
		lifeEventID = sql.NullString{String: input.LifeEventID.Value, Valid: true}
	}

	var docID string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO documents
			(case_id, person_id, life_event_id, status_id, document_type, title,
			 issuing_authority, issue_date,
			 recorded_date, recorded_given_name, recorded_surname,
			 recorded_birth_date, recorded_birth_place, notes)
		VALUES ($1, $2, $3::uuid, $4, $5::document_type, $6,
			$7, $8::date,
			$9::date, $10, $11,
			$12::date, $13, $14)
		RETURNING id::text`,
		caseID, personID, lifeEventID, statusID, docType, title,
		nullToSQL(input.IssuingAuthority),
		nullToSQL(input.IssueDate),
		nullToSQL(input.RecordedDate),
		nullToSQL(input.RecordedGivenName),
		nullToSQL(input.RecordedSurname),
		nullToSQL(input.RecordedBirthDate),
		nullToSQL(input.RecordedBirthPlace),
		nullToSQL(input.Notes),
	).Scan(&docID)
	if err != nil {
		return nil, fmt.Errorf("insert document: %w", err)
	}

	return s.getDocument(ctx, caseID, docID)
}

func (s *store) UpdateDocument(ctx context.Context, caseID, docID string, input UpdateDocumentInput) (*Document, error) {
	var isVerifiedSet, isVerifiedValue bool
	if input.IsVerified != nil {
		isVerifiedSet = true
		isVerifiedValue = *input.IsVerified
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE documents SET
			title                = COALESCE($3, title),
			document_type        = COALESCE($4::document_type, document_type),
			issuing_authority    = CASE WHEN $5  THEN $6         ELSE issuing_authority    END,
			issue_date           = CASE WHEN $7  THEN $8::date   ELSE issue_date           END,
			recorded_date        = CASE WHEN $9  THEN $10::date  ELSE recorded_date        END,
			recorded_given_name  = CASE WHEN $11 THEN $12        ELSE recorded_given_name  END,
			recorded_surname     = CASE WHEN $13 THEN $14        ELSE recorded_surname     END,
			recorded_birth_date  = CASE WHEN $15 THEN $16::date  ELSE recorded_birth_date  END,
			recorded_birth_place = CASE WHEN $17 THEN $18        ELSE recorded_birth_place END,
			notes                = CASE WHEN $19 THEN $20        ELSE notes                END,
			is_verified          = CASE WHEN $21 THEN $22        ELSE is_verified          END,
			verified_at          = CASE
				WHEN $21 AND $22 = TRUE  THEN NOW()
				WHEN $21 AND $22 = FALSE THEN NULL
				ELSE verified_at
			END,
			updated_at = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		docID, caseID,
		input.Title,
		input.DocumentType,
		input.IssuingAuthority.Set, nullToSQL(input.IssuingAuthority),
		input.IssueDate.Set, nullToSQL(input.IssueDate),
		input.RecordedDate.Set, nullToSQL(input.RecordedDate),
		input.RecordedGivenName.Set, nullToSQL(input.RecordedGivenName),
		input.RecordedSurname.Set, nullToSQL(input.RecordedSurname),
		input.RecordedBirthDate.Set, nullToSQL(input.RecordedBirthDate),
		input.RecordedBirthPlace.Set, nullToSQL(input.RecordedBirthPlace),
		input.Notes.Set, nullToSQL(input.Notes),
		isVerifiedSet, isVerifiedValue,
	)
	if err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}

	return s.getDocument(ctx, caseID, docID)
}

func (s *store) DeleteDocument(ctx context.Context, caseID, docID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE documents SET deleted_at = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		docID, caseID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete document: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *store) TransitionStatus(ctx context.Context, caseID, docID, statusKey string) (*Document, error) {
	var statusID string
	if err := s.db.QueryRowContext(ctx,
		`SELECT id::text FROM document_statuses WHERE system_key::text = $1`,
		statusKey,
	).Scan(&statusID); err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE documents SET status_id = $3, updated_at = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		docID, caseID, statusID,
	)
	if err != nil {
		return nil, fmt.Errorf("transition status: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}

	return s.getDocument(ctx, caseID, docID)
}

func (s *store) ReassignDocument(ctx context.Context, caseID, docID string, input ReassignDocumentInput) (*Document, error) {
	if err := s.requirePersonInCase(ctx, caseID, input.PersonID); err != nil {
		return nil, err
	}
	if input.LifeEventID.Set && input.LifeEventID.Valid {
		if err := s.requireLifeEventInCase(ctx, caseID, input.LifeEventID.Value); err != nil {
			return nil, err
		}
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE documents SET
			person_id     = $3,
			life_event_id = CASE WHEN $4 THEN $5::uuid ELSE life_event_id END,
			updated_at    = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		docID, caseID,
		input.PersonID,
		input.LifeEventID.Set, nullToSQL(input.LifeEventID),
	)
	if err != nil {
		return nil, fmt.Errorf("reassign document: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}

	return s.getDocument(ctx, caseID, docID)
}

func (s *store) getDocument(ctx context.Context, caseID, docID string) (*Document, error) {
	row := s.db.QueryRowContext(ctx, `
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
		WHERE d.id = $1 AND d.case_id = $2 AND d.deleted_at IS NULL`,
		docID, caseID,
	)
	d, err := scanDocument(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return d, nil
}

func (s *store) requirePersonInCase(ctx context.Context, caseID, personID string) error {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM people WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		personID, caseID,
	).Scan(&count); err != nil {
		return fmt.Errorf("verify person: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *store) requireLifeEventInCase(ctx context.Context, caseID, lifeEventID string) error {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM life_events WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		lifeEventID, caseID,
	).Scan(&count); err != nil {
		return fmt.Errorf("verify life event: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func scanDocument(scan func(...any) error) (*Document, error) {
	var d Document
	var lifeEventID, issuingAuthority, issueDate sql.NullString
	var recordedDate, recordedGivenName, recordedSurname sql.NullString
	var recordedBirthDate, recordedBirthPlace, notes sql.NullString
	var statusKey sql.NullString
	var verifiedAt sql.NullTime

	if err := scan(
		&d.ID, &d.CaseID, &d.PersonID, &lifeEventID,
		&d.DocumentType, &d.Title,
		&issuingAuthority, &issueDate,
		&recordedDate, &recordedGivenName, &recordedSurname,
		&recordedBirthDate, &recordedBirthPlace,
		&d.IsVerified, &verifiedAt, &notes,
		&d.Status, &statusKey, &d.ProgressBucket,
		&d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return nil, err
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

	return &d, nil
}

func nullToSQL(f NullableField) sql.NullString {
	return sql.NullString{String: f.Value, Valid: f.Valid}
}

func setString(dest **string, ns sql.NullString) {
	if ns.Valid {
		s := ns.String
		*dest = &s
	}
}
