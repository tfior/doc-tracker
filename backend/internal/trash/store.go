// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package trash

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tfior/doc-tracker/platform"
)

type Store interface {
	GetCaseTrash(ctx context.Context, caseID string) (*CaseTrash, error)
	GetGlobalTrash(ctx context.Context) (*GlobalTrash, error)
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) GetCaseTrash(ctx context.Context, caseID string) (*CaseTrash, error) {
	t := &CaseTrash{
		People:     []TrashedPerson{},
		LifeEvents: []TrashedLifeEvent{},
		Documents:  []TrashedDocument{},
		ClaimLines: []TrashedClaimLine{},
	}

	rows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT id::text, case_id::text, first_name, last_name, created_at, updated_at, deleted_at
		FROM people WHERE case_id = $1 AND deleted_at IS NOT NULL
		ORDER BY deleted_at DESC`, caseID)
	if err != nil {
		return nil, fmt.Errorf("query trashed people: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p TrashedPerson
		if err := rows.Scan(&p.ID, &p.CaseID, &p.FirstName, &p.LastName, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed person: %w", err)
		}
		t.People = append(t.People, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed people: %w", err)
	}

	leRows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT id::text, case_id::text, person_id::text, event_type::text,
		       event_date::text, created_at, updated_at, deleted_at
		FROM life_events WHERE case_id = $1 AND deleted_at IS NOT NULL
		ORDER BY deleted_at DESC`, caseID)
	if err != nil {
		return nil, fmt.Errorf("query trashed life events: %w", err)
	}
	defer leRows.Close()
	for leRows.Next() {
		var le TrashedLifeEvent
		var eventDate sql.NullString
		if err := leRows.Scan(&le.ID, &le.CaseID, &le.PersonID, &le.EventType, &eventDate, &le.CreatedAt, &le.UpdatedAt, &le.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed life event: %w", err)
		}
		if eventDate.Valid {
			le.EventDate = &eventDate.String
		}
		t.LifeEvents = append(t.LifeEvents, le)
	}
	if err := leRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed life events: %w", err)
	}

	docRows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT id::text, case_id::text, person_id::text, document_type::text, title,
		       created_at, updated_at, deleted_at
		FROM documents WHERE case_id = $1 AND deleted_at IS NOT NULL
		ORDER BY deleted_at DESC`, caseID)
	if err != nil {
		return nil, fmt.Errorf("query trashed documents: %w", err)
	}
	defer docRows.Close()
	for docRows.Next() {
		var d TrashedDocument
		if err := docRows.Scan(&d.ID, &d.CaseID, &d.PersonID, &d.DocumentType, &d.Title, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed document: %w", err)
		}
		t.Documents = append(t.Documents, d)
	}
	if err := docRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed documents: %w", err)
	}

	clRows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT id::text, case_id::text, root_person_id::text, status::text,
		       created_at, updated_at, deleted_at
		FROM claim_lines WHERE case_id = $1 AND deleted_at IS NOT NULL
		ORDER BY deleted_at DESC`, caseID)
	if err != nil {
		return nil, fmt.Errorf("query trashed claim lines: %w", err)
	}
	defer clRows.Close()
	for clRows.Next() {
		var cl TrashedClaimLine
		if err := clRows.Scan(&cl.ID, &cl.CaseID, &cl.RootPersonID, &cl.Status, &cl.CreatedAt, &cl.UpdatedAt, &cl.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed claim line: %w", err)
		}
		t.ClaimLines = append(t.ClaimLines, cl)
	}
	if err := clRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed claim lines: %w", err)
	}

	return t, nil
}

func (s *store) GetGlobalTrash(ctx context.Context) (*GlobalTrash, error) {
	t := &GlobalTrash{
		Cases:      []TrashedCase{},
		People:     []TrashedPerson{},
		LifeEvents: []TrashedLifeEvent{},
		Documents:  []TrashedDocument{},
		ClaimLines: []TrashedClaimLine{},
	}

	caseRows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT id::text, title, status::text, created_at, updated_at, deleted_at
		FROM cases WHERE deleted_at IS NOT NULL
		ORDER BY deleted_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query trashed cases: %w", err)
	}
	defer caseRows.Close()
	for caseRows.Next() {
		var c TrashedCase
		if err := caseRows.Scan(&c.ID, &c.Title, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed case: %w", err)
		}
		t.Cases = append(t.Cases, c)
	}
	if err := caseRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed cases: %w", err)
	}

	// For entity types: only include entities whose parent case is not trashed.
	personRows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT p.id::text, p.case_id::text, p.first_name, p.last_name,
		       p.created_at, p.updated_at, p.deleted_at
		FROM people p
		JOIN cases c ON c.id = p.case_id
		WHERE p.deleted_at IS NOT NULL AND c.deleted_at IS NULL
		ORDER BY p.deleted_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query trashed people globally: %w", err)
	}
	defer personRows.Close()
	for personRows.Next() {
		var p TrashedPerson
		if err := personRows.Scan(&p.ID, &p.CaseID, &p.FirstName, &p.LastName, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed person: %w", err)
		}
		t.People = append(t.People, p)
	}
	if err := personRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed people globally: %w", err)
	}

	leRows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT le.id::text, le.case_id::text, le.person_id::text, le.event_type::text,
		       le.event_date::text, le.created_at, le.updated_at, le.deleted_at
		FROM life_events le
		JOIN cases c ON c.id = le.case_id
		WHERE le.deleted_at IS NOT NULL AND c.deleted_at IS NULL
		ORDER BY le.deleted_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query trashed life events globally: %w", err)
	}
	defer leRows.Close()
	for leRows.Next() {
		var le TrashedLifeEvent
		var eventDate sql.NullString
		if err := leRows.Scan(&le.ID, &le.CaseID, &le.PersonID, &le.EventType, &eventDate, &le.CreatedAt, &le.UpdatedAt, &le.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed life event globally: %w", err)
		}
		if eventDate.Valid {
			le.EventDate = &eventDate.String
		}
		t.LifeEvents = append(t.LifeEvents, le)
	}
	if err := leRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed life events globally: %w", err)
	}

	docRows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT d.id::text, d.case_id::text, d.person_id::text, d.document_type::text, d.title,
		       d.created_at, d.updated_at, d.deleted_at
		FROM documents d
		JOIN cases c ON c.id = d.case_id
		WHERE d.deleted_at IS NOT NULL AND c.deleted_at IS NULL
		ORDER BY d.deleted_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query trashed documents globally: %w", err)
	}
	defer docRows.Close()
	for docRows.Next() {
		var d TrashedDocument
		if err := docRows.Scan(&d.ID, &d.CaseID, &d.PersonID, &d.DocumentType, &d.Title, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed document globally: %w", err)
		}
		t.Documents = append(t.Documents, d)
	}
	if err := docRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed documents globally: %w", err)
	}

	clRows, err := platform.Conn(ctx, s.db).QueryContext(ctx, `
		SELECT cl.id::text, cl.case_id::text, cl.root_person_id::text, cl.status::text,
		       cl.created_at, cl.updated_at, cl.deleted_at
		FROM claim_lines cl
		JOIN cases c ON c.id = cl.case_id
		WHERE cl.deleted_at IS NOT NULL AND c.deleted_at IS NULL
		ORDER BY cl.deleted_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query trashed claim lines globally: %w", err)
	}
	defer clRows.Close()
	for clRows.Next() {
		var cl TrashedClaimLine
		if err := clRows.Scan(&cl.ID, &cl.CaseID, &cl.RootPersonID, &cl.Status, &cl.CreatedAt, &cl.UpdatedAt, &cl.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan trashed claim line globally: %w", err)
		}
		t.ClaimLines = append(t.ClaimLines, cl)
	}
	if err := clRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trashed claim lines globally: %w", err)
	}

	return t, nil
}
