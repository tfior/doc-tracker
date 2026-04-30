package documents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("document not found")
var ErrInvalidStatus = errors.New("status does not belong to the requested phase")

var validPhases = map[string]bool{
	"official_copy": true, "amendment": true, "apostille": true, "translation": true,
}

type Store interface {
	ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error)
	ListDocumentStatuses(ctx context.Context, phase string) ([]DocumentStatus, error)
	CreateDocument(ctx context.Context, caseID, personID, docType, title string, input UpdateDocumentInput) (*Document, error)
	UpdateDocument(ctx context.Context, caseID, docID string, input UpdateDocumentInput) (*Document, error)
	DeleteDocument(ctx context.Context, caseID, docID string) error
	TransitionStatus(ctx context.Context, caseID, docID string, input TransitionStatusInput) (*Document, error)
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
			oc.id::text, oc.label, oc.phase::text, oc.progress_bucket::text,
			am.id::text, am.label, am.phase::text, am.progress_bucket::text,
			ap.id::text, ap.label, ap.phase::text, ap.progress_bucket::text,
			tr.id::text, tr.label, tr.phase::text, tr.progress_bucket::text,
			d.created_at, d.updated_at
		FROM documents d
		JOIN document_statuses oc ON oc.id = d.official_copy_status_id
		JOIN document_statuses am ON am.id = d.amendment_status_id
		JOIN document_statuses ap ON ap.id = d.apostille_status_id
		JOIN document_statuses tr ON tr.id = d.translation_status_id
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

func (s *store) ListDocumentStatuses(ctx context.Context, phase string) ([]DocumentStatus, error) {
	var rows *sql.Rows
	var err error

	if phase == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id::text, label, phase::text, is_system, progress_bucket::text
			FROM document_statuses
			ORDER BY phase, progress_bucket, label`)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id::text, label, phase::text, is_system, progress_bucket::text
			FROM document_statuses
			WHERE phase = $1 OR phase = 'any'
			ORDER BY progress_bucket, label`,
			phase)
	}
	if err != nil {
		return nil, fmt.Errorf("query document statuses: %w", err)
	}
	defer rows.Close()

	items := []DocumentStatus{}
	for rows.Next() {
		var ds DocumentStatus
		if err := rows.Scan(&ds.ID, &ds.Label, &ds.Phase, &ds.IsSystem, &ds.ProgressBucket); err != nil {
			return nil, fmt.Errorf("scan document status: %w", err)
		}
		items = append(items, ds)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate document statuses: %w", err)
	}

	return items, nil
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

	defaults, err := s.phaseDefaults(ctx)
	if err != nil {
		return nil, err
	}

	var lifeEventID sql.NullString
	if input.LifeEventID.Set && input.LifeEventID.Valid {
		lifeEventID = sql.NullString{String: input.LifeEventID.Value, Valid: true}
	}

	var docID string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO documents
			(case_id, person_id, life_event_id,
			 official_copy_status_id, amendment_status_id, apostille_status_id, translation_status_id,
			 document_type, title,
			 issuing_authority, issue_date,
			 recorded_date, recorded_given_name, recorded_surname,
			 recorded_birth_date, recorded_birth_place, notes)
		VALUES ($1, $2, $3::uuid,
			$4, $5, $6, $7,
			$8::document_type, $9,
			$10, $11::date,
			$12::date, $13, $14,
			$15::date, $16, $17)
		RETURNING id::text`,
		caseID, personID, lifeEventID,
		defaults["official_copy"], defaults["amendment"], defaults["apostille"], defaults["translation"],
		docType, title,
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

func (s *store) TransitionStatus(ctx context.Context, caseID, docID string, input TransitionStatusInput) (*Document, error) {
	// Verify the status exists and belongs to the requested phase (or is 'any').
	var statusPhase string
	if err := s.db.QueryRowContext(ctx,
		`SELECT phase::text FROM document_statuses WHERE id = $1`, input.StatusID,
	).Scan(&statusPhase); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidStatus
		}
		return nil, fmt.Errorf("get status phase: %w", err)
	}
	if statusPhase != input.Phase && statusPhase != "any" {
		return nil, ErrInvalidStatus
	}

	col := phaseColumn(input.Phase)
	result, err := s.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE documents SET %s = $3, updated_at = NOW()
			WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`, col),
		docID, caseID, input.StatusID,
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
			oc.id::text, oc.label, oc.phase::text, oc.progress_bucket::text,
			am.id::text, am.label, am.phase::text, am.progress_bucket::text,
			ap.id::text, ap.label, ap.phase::text, ap.progress_bucket::text,
			tr.id::text, tr.label, tr.phase::text, tr.progress_bucket::text,
			d.created_at, d.updated_at
		FROM documents d
		JOIN document_statuses oc ON oc.id = d.official_copy_status_id
		JOIN document_statuses am ON am.id = d.amendment_status_id
		JOIN document_statuses ap ON ap.id = d.apostille_status_id
		JOIN document_statuses tr ON tr.id = d.translation_status_id
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

// phaseDefaults returns a map of phase → default status UUID for document creation.
func (s *store) phaseDefaults(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT phase::text, id::text FROM document_statuses WHERE system_key IN
		 ('official_copy_default', 'amendment_default', 'apostille_default', 'translation_default')`)
	if err != nil {
		return nil, fmt.Errorf("get phase defaults: %w", err)
	}
	defer rows.Close()

	defaults := make(map[string]string, 4)
	for rows.Next() {
		var phase, id string
		if err := rows.Scan(&phase, &id); err != nil {
			return nil, fmt.Errorf("scan phase default: %w", err)
		}
		defaults[phase] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate phase defaults: %w", err)
	}
	if len(defaults) != 4 {
		return nil, fmt.Errorf("expected 4 phase defaults, got %d", len(defaults))
	}
	return defaults, nil
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
	var verifiedAt sql.NullTime

	if err := scan(
		&d.ID, &d.CaseID, &d.PersonID, &lifeEventID,
		&d.DocumentType, &d.Title,
		&issuingAuthority, &issueDate,
		&recordedDate, &recordedGivenName, &recordedSurname,
		&recordedBirthDate, &recordedBirthPlace,
		&d.IsVerified, &verifiedAt, &notes,
		&d.OfficialCopyStatus.ID, &d.OfficialCopyStatus.Label, &d.OfficialCopyStatus.Phase, &d.OfficialCopyStatus.ProgressBucket,
		&d.AmendmentStatus.ID, &d.AmendmentStatus.Label, &d.AmendmentStatus.Phase, &d.AmendmentStatus.ProgressBucket,
		&d.ApostilleStatus.ID, &d.ApostilleStatus.Label, &d.ApostilleStatus.Phase, &d.ApostilleStatus.ProgressBucket,
		&d.TranslationStatus.ID, &d.TranslationStatus.Label, &d.TranslationStatus.Phase, &d.TranslationStatus.ProgressBucket,
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
	if verifiedAt.Valid {
		t := verifiedAt.Time.UTC()
		d.VerifiedAt = &t
	}

	return &d, nil
}

// phaseColumn returns the documents column name for a given phase.
// Caller must validate phase before calling.
func phaseColumn(phase string) string {
	switch phase {
	case "official_copy":
		return "official_copy_status_id"
	case "amendment":
		return "amendment_status_id"
	case "apostille":
		return "apostille_status_id"
	default: // "translation"
		return "translation_status_id"
	}
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
