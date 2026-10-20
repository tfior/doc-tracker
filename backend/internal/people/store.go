// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package people

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/tfior/doc-tracker/platform"
)

var (
	ErrNotFound = fmt.Errorf("person %w", platform.ErrNotFound)
)

const (
	pgFKViolation     = "23503"
	pgUniqueViolation = "23505"
)

type Store interface {
	LockActive(ctx context.Context, caseID, id string) error
	ListPeople(ctx context.Context, caseID string, page, perPage int) ([]Person, int, error)
	CreatePerson(ctx context.Context, caseID, firstName, lastName string, input UpdatePersonInput) (*Person, error)
	UpdatePerson(ctx context.Context, caseID, personID string, input UpdatePersonInput) (*Person, error)
	DeletePerson(ctx context.Context, caseID, personID string) error
	LockPeople(ctx context.Context, ids ...string) error
	AddParent(ctx context.Context, caseID, personID, parentID string) (*Relationship, error)
	RemoveParent(ctx context.Context, caseID, personID, parentID string) error
	RequireActive(ctx context.Context, caseID, personID string) error
	ChildWithFullParents(ctx context.Context, caseID, personID string) (string, error)
	Restore(ctx context.Context, caseID, id string) error
	PermanentDelete(ctx context.Context, caseID, id string) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListPeople(ctx context.Context, caseID string, page, perPage int) ([]Person, int, error) {
	var total int
	if err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM active_people WHERE case_id = $1`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count people: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT
			id::text, case_id::text,
			first_name, last_name,
			birth_date::text, birth_place,
			death_date::text, notes,
			created_at, updated_at,
			ARRAY(SELECT pr.parent_id::text FROM person_relationships pr JOIN active_people pa ON pa.id = pr.parent_id WHERE pr.person_id = people.id ORDER BY pr.parent_id) AS parent_ids
		FROM people
		WHERE id IN (SELECT id FROM active_people WHERE case_id = $1)
		ORDER BY birth_date ASC NULLS LAST, created_at ASC, id ASC
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
		var parentIDs pq.StringArray
		if err := rows.Scan(
			&p.ID, &p.CaseID,
			&p.FirstName, &p.LastName,
			&birthDate, &birthPlace,
			&deathDate, &notes,
			&p.CreatedAt, &p.UpdatedAt,
			&parentIDs,
		); err != nil {
			return nil, 0, fmt.Errorf("scan person: %w", err)
		}
		assignNullable(&p, birthDate, birthPlace, deathDate, notes)
		p.ParentIDs = toStringSlice(parentIDs)
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate people: %w", err)
	}

	return items, total, nil
}

func (s *store) CreatePerson(ctx context.Context, caseID, firstName, lastName string, input UpdatePersonInput) (*Person, error) {
	var id string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		INSERT INTO people (case_id, first_name, last_name, birth_date, birth_place, death_date, notes)
		VALUES ($1, $2, $3, $4::date, $5, $6::date, $7)
		RETURNING id::text`,
		caseID, firstName, lastName,
		nullableToSQL(input.BirthDate),
		nullableToSQL(input.BirthPlace),
		nullableToSQL(input.DeathDate),
		nullableToSQL(input.Notes),
	).Scan(&id)
	if err != nil {
		if isPGError(err, pgFKViolation) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("insert person: %w", err)
	}
	return s.getPerson(ctx, caseID, id)
}

func (s *store) UpdatePerson(ctx context.Context, caseID, personID string, input UpdatePersonInput) (*Person, error) {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
		UPDATE people SET
			first_name  = COALESCE($3, first_name),
			last_name   = COALESCE($4, last_name),
			birth_date  = CASE WHEN $5 THEN $6::date  ELSE birth_date  END,
			birth_place = CASE WHEN $7 THEN $8         ELSE birth_place END,
			death_date  = CASE WHEN $9 THEN $10::date  ELSE death_date  END,
			notes       = CASE WHEN $11 THEN $12        ELSE notes       END,
			updated_at  = NOW()
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		personID, caseID,
		input.FirstName,
		input.LastName,
		input.BirthDate.Set, nullableToSQL(input.BirthDate),
		input.BirthPlace.Set, nullableToSQL(input.BirthPlace),
		input.DeathDate.Set, nullableToSQL(input.DeathDate),
		input.Notes.Set, nullableToSQL(input.Notes),
	)
	if err != nil {
		return nil, fmt.Errorf("update person: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.getPerson(ctx, caseID, personID)
}

func (s *store) getPerson(ctx context.Context, caseID, personID string) (*Person, error) {
	var p Person
	var birthDate, birthPlace, deathDate, notes sql.NullString
	var parentIDs pq.StringArray

	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT
			id::text, case_id::text,
			first_name, last_name,
			birth_date::text, birth_place,
			death_date::text, notes,
			created_at, updated_at,
			ARRAY(SELECT pr.parent_id::text FROM person_relationships pr JOIN active_people pa ON pa.id = pr.parent_id WHERE pr.person_id = people.id ORDER BY pr.parent_id) AS parent_ids
		FROM people
		WHERE id = $1 AND case_id = $2 AND deleted_at IS NULL`,
		personID, caseID,
	).Scan(
		&p.ID, &p.CaseID,
		&p.FirstName, &p.LastName,
		&birthDate, &birthPlace,
		&deathDate, &notes,
		&p.CreatedAt, &p.UpdatedAt,
		&parentIDs,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get person: %w", err)
	}
	assignNullable(&p, birthDate, birthPlace, deathDate, notes)
	p.ParentIDs = toStringSlice(parentIDs)
	return &p, nil
}

func (s *store) DeletePerson(ctx context.Context, caseID, personID string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx, `
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

// AddParent links a parent to a person. The caller must already hold locks on
// both people (LockPeople), so the checks and the insert see the same state.
func (s *store) AddParent(ctx context.Context, caseID, personID, parentID string) (*Relationship, error) {
	conn := platform.Conn(ctx, s.db)

	// A loop would form if the new parent already descends from the person.
	// Every link counts, including those to trashed people, because a trashed
	// person can be restored.
	var loop bool
	if err := conn.QueryRowContext(ctx, `
		WITH RECURSIVE descendants(id) AS (
			SELECT $1::uuid
			UNION
			SELECT pr.person_id FROM person_relationships pr
			JOIN descendants d ON pr.parent_id = d.id
		)
		SELECT EXISTS (SELECT 1 FROM descendants WHERE id = $2::uuid)`,
		personID, parentID,
	).Scan(&loop); err != nil {
		return nil, fmt.Errorf("check for loop: %w", err)
	}
	if loop {
		return nil, platform.Conflict("This link would make a person their own ancestor.")
	}

	// Only active parents count towards the limit.
	var parentCount int
	if err := conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM person_relationships pr
		JOIN active_people ap ON ap.id = pr.parent_id
		WHERE pr.person_id = $1`,
		personID,
	).Scan(&parentCount); err != nil {
		return nil, fmt.Errorf("count parents: %w", err)
	}
	if parentCount >= 2 {
		return nil, platform.Conflict("A person can have at most two parents. Remove a parent link first.")
	}

	_, err := conn.ExecContext(ctx,
		`INSERT INTO person_relationships (person_id, parent_id, case_id) VALUES ($1, $2, $3)`,
		personID, parentID, caseID,
	)
	if err != nil {
		if isPGError(err, pgUniqueViolation) {
			return nil, platform.Conflict("This parent link already exists.")
		}
		return nil, fmt.Errorf("insert relationship: %w", err)
	}
	return &Relationship{PersonID: personID, ParentID: parentID, CaseID: caseID}, nil
}

func (s *store) RemoveParent(ctx context.Context, caseID, personID, parentID string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
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

func toStringSlice(a pq.StringArray) []string {
	if a == nil {
		return []string{}
	}
	return []string(a)
}

func nullableToSQL(f platform.NullString) sql.NullString {
	return sql.NullString{String: f.Value, Valid: f.Valid}
}

func isPGError(err error, code string) bool {
	var pgErr *pq.Error
	return errors.As(err, &pgErr) && string(pgErr.Code) == code
}

// RequireActive returns ErrNotFound unless the person is active: not in the
// trash and not in a trashed case. It locks the person and their case until
// the transaction ends, so neither can be trashed while a write that depends
// on them is in progress.
func (s *store) RequireActive(ctx context.Context, caseID, personID string) error {
	var id string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx,
		`SELECT id::text FROM active_people WHERE id = $1 AND case_id = $2 FOR SHARE`,
		personID, caseID,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("verify person: %w", err)
	}
	return nil
}

// ChildWithFullParents returns the name of a child of the trashed person who
// already has two active parents, so restoring the person would give them a
// third. It returns "" when there is none.
func (s *store) ChildWithFullParents(ctx context.Context, caseID, personID string) (string, error) {
	// Lock the children, as adding a parent does, so two restores running at
	// once cannot both pass this check for the same child.
	if _, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`SELECT id FROM people WHERE id IN (SELECT person_id FROM person_relationships WHERE parent_id = $1)
		 ORDER BY id FOR UPDATE`, personID); err != nil {
		return "", fmt.Errorf("lock children: %w", err)
	}
	var name string
	err := platform.Conn(ctx, s.db).QueryRowContext(ctx, `
		SELECT c.first_name || ' ' || c.last_name
		FROM person_relationships pr
		JOIN people c ON c.id = pr.person_id
		WHERE pr.parent_id = $1 AND pr.case_id = $2
		  AND (SELECT COUNT(*) FROM person_relationships pr2
		       JOIN active_people ap ON ap.id = pr2.parent_id
		       WHERE pr2.person_id = pr.person_id AND pr2.parent_id <> $1) >= 2
		ORDER BY c.last_name, c.first_name
		LIMIT 1`,
		personID, caseID,
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("check children's parents: %w", err)
	}
	return name, nil
}

func (s *store) Restore(ctx context.Context, caseID, id string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`UPDATE people SET deleted_at = NULL WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		id, caseID)
	if err != nil {
		return fmt.Errorf("restore person: %w", err)
	}
	return platform.RequireOneRow(result, ErrNotFound)
}

func (s *store) PermanentDelete(ctx context.Context, caseID, id string) error {
	result, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`DELETE FROM people WHERE id = $1 AND case_id = $2 AND deleted_at IS NOT NULL`,
		id, caseID)
	if err != nil {
		return fmt.Errorf("permanent delete person: %w", err)
	}
	return platform.RequireOneRow(result, ErrNotFound)
}

// LockActive takes an exclusive lock on a row about to be changed, then checks
// that it is active. Taking the exclusive lock first means simultaneous writes
// to the same row wait for each other instead of deadlocking.
func (s *store) LockActive(ctx context.Context, caseID, id string) error {
	if _, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`SELECT id FROM people WHERE id = $1 FOR NO KEY UPDATE`, id); err != nil {
		return fmt.Errorf("lock people: %w", err)
	}
	return s.RequireActive(ctx, caseID, id)
}

// LockPeople takes exclusive locks on people, always in the same order, so that
// concurrent changes to family links involving any of them run one at a time.
// Without this, two requests could each see one parent and both add a second,
// or two opposite links could together form a loop.
func (s *store) LockPeople(ctx context.Context, ids ...string) error {
	if _, err := platform.Conn(ctx, s.db).ExecContext(ctx,
		`SELECT id FROM people WHERE id = ANY($1::uuid[]) ORDER BY id FOR UPDATE`,
		pq.Array(ids)); err != nil {
		return fmt.Errorf("lock people: %w", err)
	}
	return nil
}
