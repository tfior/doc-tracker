package people

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	ListPeople(ctx context.Context, caseID string, page, perPage int) ([]Person, int, error)
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
		`SELECT COUNT(*) FROM people WHERE case_id = $1`, caseID,
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
		WHERE case_id = $1
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
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate people: %w", err)
	}

	return items, total, nil
}
