package people

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var (
	ErrNotFound = errors.New("person not found")
	ErrConflict = errors.New("conflict")
)

const (
	pgFKViolation     = "23503"
	pgUniqueViolation = "23505"
)

type Store interface {
	ListPeople(ctx context.Context, caseID string, page, perPage int) ([]Person, int, error)
	CreatePerson(ctx context.Context, caseID, firstName, lastName string, input UpdatePersonInput) (*Person, error)
	UpdatePerson(ctx context.Context, caseID, personID string, input UpdatePersonInput) (*Person, error)
	DeletePerson(ctx context.Context, caseID, personID string) error
	AddParent(ctx context.Context, caseID, personID, parentID string) (*Relationship, error)
	RemoveParent(ctx context.Context, caseID, personID, parentID string) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListPeople(ctx context.Context, caseID string, page, perPage int) ([]Person, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM people WHERE case_id = $1 AND deleted_at IS NULL`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count people: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id::text, case_id::text,
			first_name, last_name,
			birth_date::text, birth_place,
			death_date::text, notes,
			created_at, updated_at
		FROM people
		WHERE case_id = $1 AND deleted_at IS NULL
		ORDER BY birth_date ASC NULLS LAST
		LIMIT $2 OFFSET $3`,
		caseID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query people: %w", err)
	}
	defer rows.Close()

	items := []Person{}
	for rows.Next() {
		var p Person
		var birthDate, birthPlace, deathDate, notes sql.NullString
		if err := rows.Scan(
			&p.ID, &p.CaseID,
			&p.FirstName, &p.LastName,
			&birthDate, &birthPlace,
			&deathDate, &notes,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan person: %w", err)
		}
		assignNullable(&p, birthDate, birthPlace, deathDate, notes)
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate people: %w", err)
	}

	return items, total, nil
}

func (s *store) CreatePerson(ctx context.Context, caseID, firstName, lastName string, input UpdatePersonInput) (*Person, error) {
	var p Person
	var birthDate, birthPlace, deathDate, notes sql.NullString

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO people (case_id, first_name, last_name, birth_date, birth_place, death_date, notes)
		VALUES ($1, $2, $3, $4::date, $5, $6::date, $7)
		RETURNING
			id::text, case_id::text,
			first_name, last_name,
			birth_date::text, birth_place,
			death_date::text, notes,
			created_at, updated_at`,
		caseID, firstName, lastName,
		nullableToSQL(input.BirthDate),
		nullableToSQL(input.BirthPlace),
		nullableToSQL(input.DeathDate),
		nullableToSQL(input.Notes),
	).Scan(
		&p.ID, &p.CaseID,
		&p.FirstName, &p.LastName,
		&birthDate, &birthPlace,
		&deathDate, &notes,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if isPGError(err, pgFKViolation) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("insert person: %w", err)
	}
	assignNullable(&p, birthDate, birthPlace, deathDate, notes)
	return &p, nil
}

func (s *store) UpdatePerson(ctx context.Context, caseID, personID string, input UpdatePersonInput) (*Person, error) {
	var p Person
	var birthDate, birthPlace, deathDate, notes sql.NullString

	err := s.db.QueryRowContext(ctx, `
		UPDATE people SET
			first_name  = COALESCE($3, first_name),
			last_name   = COALESCE($4, last_name),
			birth_date  = CASE WHEN $5 THEN $6::date  ELSE birth_date  END,
			birth_place = CASE WHEN $7 THEN $8         ELSE birth_place END,
			death_date  = CASE WHEN $9 THEN $10::date  ELSE death_date  END,
			notes       = CASE WHEN $11 THEN $12        ELSE notes       END,
			updated_at  = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL
		RETURNING
			id::text, case_id::text,
			first_name, last_name,
			birth_date::text, birth_place,
			death_date::text, notes,
			created_at, updated_at`,
		personID, caseID,
		input.FirstName,
		input.LastName,
		input.BirthDate.Set, nullableToSQL(input.BirthDate),
		input.BirthPlace.Set, nullableToSQL(input.BirthPlace),
		input.DeathDate.Set, nullableToSQL(input.DeathDate),
		input.Notes.Set, nullableToSQL(input.Notes),
	).Scan(
		&p.ID, &p.CaseID,
		&p.FirstName, &p.LastName,
		&birthDate, &birthPlace,
		&deathDate, &notes,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update person: %w", err)
	}
	assignNullable(&p, birthDate, birthPlace, deathDate, notes)
	return &p, nil
}

func (s *store) DeletePerson(ctx context.Context, caseID, personID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE people SET deleted_at = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		personID, caseID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete person: %w", err)
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

func (s *store) AddParent(ctx context.Context, caseID, personID, parentID string) (*Relationship, error) {
	// Verify both person and parent exist and belong to the case.
	var count int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM people
		WHERE id = ANY($1::uuid[]) AND case_id = $2 AND deleted_at IS NULL`,
		pq.Array([]string{personID, parentID}), caseID,
	).Scan(&count); err != nil {
		return nil, fmt.Errorf("verify people: %w", err)
	}
	// personID and parentID are distinct (validated before this call), so we
	// expect exactly 2 rows if both belong to the case.
	if count < 2 {
		return nil, ErrNotFound
	}

	// Enforce max 2 parents.
	var parentCount int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM person_relationships WHERE person_id = $1`,
		personID,
	).Scan(&parentCount); err != nil {
		return nil, fmt.Errorf("count parents: %w", err)
	}
	if parentCount >= 2 {
		return nil, ErrConflict
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO person_relationships (person_id, parent_id, case_id) VALUES ($1, $2, $3)`,
		personID, parentID, caseID,
	)
	if err != nil {
		if isPGError(err, pgUniqueViolation) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert relationship: %w", err)
	}
	return &Relationship{PersonID: personID, ParentID: parentID, CaseID: caseID}, nil
}

func (s *store) RemoveParent(ctx context.Context, caseID, personID, parentID string) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM person_relationships WHERE person_id = $1 AND parent_id = $2 AND case_id = $3`,
		personID, parentID, caseID,
	)
	if err != nil {
		return fmt.Errorf("delete relationship: %w", err)
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

// assignNullable maps scanned NullString values onto a Person pointer.
func assignNullable(p *Person, birthDate, birthPlace, deathDate, notes sql.NullString) {
	if birthDate.Valid {
		p.BirthDate = &birthDate.String
	}
	if birthPlace.Valid {
		p.BirthPlace = &birthPlace.String
	}
	if deathDate.Valid {
		p.DeathDate = &deathDate.String
	}
	if notes.Valid {
		p.Notes = &notes.String
	}
}

func nullableToSQL(f NullableField) sql.NullString {
	return sql.NullString{String: f.Value, Valid: f.Valid}
}

func isPGError(err error, code string) bool {
	var pgErr *pq.Error
	return errors.As(err, &pgErr) && string(pgErr.Code) == code
}
