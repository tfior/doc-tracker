package activitylog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type Store interface {
	Insert(ctx context.Context, p InsertParams) error
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) Insert(ctx context.Context, p InsertParams) error {
	var changesJSON interface{}
	if len(p.Changes) > 0 {
		b, err := json.Marshal(p.Changes)
		if err != nil {
			return fmt.Errorf("marshal changes: %w", err)
		}
		changesJSON = string(b)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO activity_logs
			(case_id, user_id, action, entity_type, entity_id, entity_name, changes)
		VALUES ($1, $2, $3::activity_action, $4::activity_entity_type, $5::uuid, $6, $7::jsonb)`,
		p.CaseID, p.UserID, p.Action, p.EntityType, p.EntityID, p.EntityName, changesJSON,
	)
	if err != nil {
		return fmt.Errorf("insert activity log: %w", err)
	}
	return nil
}
