package lifeevents

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	ListLifeEvents(ctx context.Context, caseID string, page, perPage int) ([]LifeEvent, int, error)
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) ListLifeEvents(ctx context.Context, caseID string, page, perPage int) ([]LifeEvent, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM life_events WHERE case_id = $1`, caseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count life events: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			le.id::text, le.case_id::text, le.person_id::text,
			le.event_type::text,
			le.event_date::text, le.event_place,
			le.spouse_name, le.spouse_birth_date::text, le.spouse_birth_place,
			le.notes,
			COUNT(d.id) > 0 AS has_documents,
			le.created_at, le.updated_at
		FROM life_events le
		LEFT JOIN documents d ON d.life_event_id = le.id
		WHERE le.case_id = $1
		GROUP BY le.id
		ORDER BY le.event_date ASC NULLS LAST, le.created_at ASC
		LIMIT $2 OFFSET $3`,
		caseID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query life events: %w", err)
	}
	defer rows.Close()

	items := []LifeEvent{}
	for rows.Next() {
		var le LifeEvent
		var eventDate, eventPlace, spouseName, spouseBirthDate, spouseBirthPlace, notes sql.NullString
		if err := rows.Scan(
			&le.ID, &le.CaseID, &le.PersonID,
			&le.EventType,
			&eventDate, &eventPlace,
			&spouseName, &spouseBirthDate, &spouseBirthPlace,
			&notes,
			&le.HasDocuments,
			&le.CreatedAt, &le.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan life event: %w", err)
		}
		if eventDate.Valid {
			le.EventDate = &eventDate.String
		}
		if eventPlace.Valid {
			le.EventPlace = &eventPlace.String
		}
		if spouseName.Valid {
			le.SpouseName = &spouseName.String
		}
		if spouseBirthDate.Valid {
			le.SpouseBirthDate = &spouseBirthDate.String
		}
		if spouseBirthPlace.Valid {
			le.SpouseBirthPlace = &spouseBirthPlace.String
		}
		if notes.Valid {
			le.Notes = &notes.String
		}
		items = append(items, le)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate life events: %w", err)
	}

	return items, total, nil
}
