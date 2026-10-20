// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/tfior/doc-tracker/platform"
)

var ErrNotFound = fmt.Errorf("document %w", platform.ErrNotFound)
var ErrInvalidStatus = errors.New("status does not belong to the requested phase")

var validPhases = map[string]bool{
	"official_copy": true, "amendment": true, "apostille": true, "translation": true,
}

type Store interface {
	IDsForLifeEvent(ctx context.Context, caseID, eventID string) ([]string, error)
	IDsForPerson(ctx context.Context, caseID, personID string) ([]string, error)
	Get(ctx context.Context, caseID, id string) (*Document, error)
	LogName(ctx context.Context, caseID, id string) (string, error)
	LockActive(ctx context.Context, caseID, id string) (personID string, err error)
	ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error)
	ListDocumentStatuses(ctx context.Context, phase string) ([]DocumentStatus, error)
	CreateDocument(ctx context.Context, caseID, personID, docType, title string, input UpdateDocumentInput) (*Document, error)
	UpdateDocument(ctx context.Context, caseID, docID string, input UpdateDocumentInput) (*Document, error)
	DeleteDocument(ctx context.Context, caseID, docID string) error
	TransitionStatus(ctx context.Context, caseID, docID string, input TransitionStatusInput) (*Document, error)
	ReassignDocument(ctx context.Context, caseID, docID string, input ReassignDocumentInput) (*Document, error)
	RequireActive(ctx context.Context, caseID, docID string) (personID string, err error)
	TrashedParents(ctx context.Context, caseID, docID string) (personID string, lifeEventID *string, err error)
	MoveEventDocuments(ctx context.Context, caseID, eventID, personID string) ([]MovedDocument, error)
	LockEventDocuments(ctx context.Context, caseID, eventID string) error
	Restore(ctx context.Context, caseID, id string) error
	PermanentDelete(ctx context.Context, caseID, id string) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error) {
	var total int
	if err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM active_documents WHERE case_id = $1`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count documents: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
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
		WHERE d.id IN (SELECT id FROM active_documents WHERE case_id = $1)
		ORDER BY d.created_at ASC, d.id ASC
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
		rows, err = platform.Conn(ctx, s.db).QueryContext(ctx, `
			SELECT id::text, label, phase::text, is_system, progress_bucket::text
			FROM document_statuses
			ORDER BY phase, progress_bucket, label`)
	} else {
		rows, err = platform.Conn(ctx, s.db).QueryContext(ctx, `
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

	defaults, err := s.phaseDefaults(ctx)
	if err != nil {
		return nil, err
	}

	var lifeEventID sql.NullString
	if input.LifeEventID.Set && input.LifeEventID.Valid {
		lifeEventID = sql.NullString{String: input.LifeEventID.Value, Valid: true}
	}

	var docID string
	err = platform.Conn(ctx, s.db).QueryRowContext(ctx, `
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

	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
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
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
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
	if err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
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
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
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

	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
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
	row := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
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
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx,
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

func nullToSQL(f platform.NullString) sql.NullString {
	return sql.NullString{String: f.Value, Valid: f.Valid}
}

func setString(dest **string, ns sql.NullString) {
	if ns.Valid {
		s := ns.String
		*dest = &s
	}
}

func (s *store) Restore(ctx context.Context, caseID, id string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`UPDATE documents SET deleted_at = NULL WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		id, caseID)
	if err != nil {
		return fmt.Errorf("restore document: %w", err)
	}
	return platform.RequireOneRow(result, ErrNotFound)
}

func (s *store) PermanentDelete(ctx context.Context, caseID, id string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`DELETE FROM documents WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		id, caseID)
	if err != nil {
		return fmt.Errorf("permanent delete document: %w", err)
	}
	return platform.RequireOneRow(result, ErrNotFound)
}

// RequireActive returns the document's person, or ErrNotFound unless the
// document is active: not in the trash and not beneath a trashed life event,
// person or case. It locks the document, its person and its case until the
// transaction ends.
func (s *store) RequireActive(ctx context.Context, caseID, docID string) (string, error) {
	conn := platform.Conn(ctx, s.db)
	var personID string
	var lifeEventID sql.NullString
	err := conn.QueryRowContext(ctx,
		`SELECT person_id::text, life_event_id::text FROM active_documents
		 WHERE id = $1 AND case_id = $2 FOR SHARE`,
		docID, caseID,
	).Scan(&personID, &lifeEventID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("verify document: %w", err)
	}
	// The view checks the life event without locking it, so lock it here: the
	// event cannot then be trashed while this document is being changed.
	if lifeEventID.Valid {
		var id string
		err := conn.QueryRowContext(ctx,
			`SELECT id::text FROM life_events WHERE id = $1 AND deleted_at IS NULL FOR SHARE`,
			lifeEventID.String,
		).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		if err != nil {
			return "", fmt.Errorf("verify document's life event: %w", err)
		}
	}
	return personID, nil
}

// TrashedParents returns the person and life event of a document that is in
// the trash, or ErrNotFound if the document is not in the trash.
func (s *store) TrashedParents(ctx context.Context, caseID, docID string) (string, *string, error) {
	var personID string
	var lifeEventID sql.NullString
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT person_id::text, life_event_id::text FROM documents
		 WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		docID, caseID,
	).Scan(&personID, &lifeEventID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("get trashed document: %w", err)
	}
	var ev *string
	setString(&ev, lifeEventID)
	return personID, ev, nil
}

// MoveEventDocuments gives every document under a life event, including any in
// the trash, the event's new person.
func (s *store) MoveEventDocuments(ctx context.Context, caseID, eventID, personID string) ([]MovedDocument, error) {
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		UPDATE documents d SET person_id = $3, updated_at = NOW()
		FROM (SELECT id, person_id FROM documents
		      WHERE life_event_id = $1 AND case_id = $2 AND person_id <> $3) old
		WHERE d.id = old.id
		RETURNING d.id::text, d.title, old.person_id::text`,
		eventID, caseID, personID,
	)
	if err != nil {
		return nil, fmt.Errorf("move event documents: %w", err)
	}
	defer rows.Close()
	moved := []MovedDocument{}
	for rows.Next() {
		var m MovedDocument
		if err := rows.Scan(&m.ID, &m.Title, &m.FromPersonID); err != nil {
			return nil, fmt.Errorf("scan moved document: %w", err)
		}
		moved = append(moved, m)
	}
	return moved, rows.Err()
}

// LockActive takes an exclusive lock on a row about to be changed, then checks
// that it is active. Taking the exclusive lock first means simultaneous writes
// to the same row wait for each other instead of deadlocking.
func (s *store) LockActive(ctx context.Context, caseID, id string) (string, error) {
	if _, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`SELECT id FROM documents WHERE id = $1 FOR NO KEY UPDATE`, id); err != nil {
		return "", fmt.Errorf("lock documents: %w", err)
	}
	return s.RequireActive(ctx, caseID, id)
}

// LockEventDocuments takes an exclusive lock on every document under a life
// event. Anything that changes both an event and its documents locks the
// documents first, the same order a document write uses, so the two cannot
// deadlock.
func (s *store) LockEventDocuments(ctx context.Context, caseID, eventID string) error {
	if _, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`SELECT id FROM documents WHERE life_event_id = $1 AND case_id = $2 ORDER BY id FOR NO KEY UPDATE`,
		eventID, caseID); err != nil {
		return fmt.Errorf("lock event documents: %w", err)
	}
	return nil
}

// Get returns an active record, for the "before" side of an update's log entry.
func (s *store) Get(ctx context.Context, caseID, id string) (*Document, error) {
	return s.getDocument(ctx, caseID, id)
}

// LogName is the record's name for the activity log. It reads trashed records
// too, so restores and permanent deletes can be named.
func (s *store) LogName(ctx context.Context, caseID, id string) (string, error) {
	var name string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT title FROM documents WHERE id = $1 AND case_id = $2`, id, caseID,
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("log name: %w", err)
	}
	return name, nil
}

// IDsForPerson returns the IDs of a person's documents, trashed ones included.
func (s *store) IDsForPerson(ctx context.Context, caseID, personID string) ([]string, error) {
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx,
		`SELECT id::text FROM documents WHERE case_id = $1 AND person_id = $2 ORDER BY id`, caseID, personID)
	if err != nil {
		return nil, fmt.Errorf("IDsForPerson: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("IDsForPerson: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// IDsForLifeEvent returns the IDs of a life event's documents, trashed ones included.
func (s *store) IDsForLifeEvent(ctx context.Context, caseID, eventID string) ([]string, error) {
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx,
		`SELECT id::text FROM documents WHERE case_id = $1 AND life_event_id = $2 ORDER BY id`, caseID, eventID)
	if err != nil {
		return nil, fmt.Errorf("IDsForLifeEvent: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("IDsForLifeEvent: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
