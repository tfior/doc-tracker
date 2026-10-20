-- Restores and permanent deletes are logged (domain model, Activity Log).
ALTER TYPE activity_action ADD VALUE 'restored';
ALTER TYPE activity_action ADD VALUE 'permanently_deleted';

-- A permanent record that a case was permanently deleted (domain model,
-- CaseDeletionRecord). It has no foreign key to cases, so it survives the
-- deletion, and it holds no personal data beyond the case title. Owned by the
-- trash module.
CREATE TABLE case_deletion_records (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id           UUID        NOT NULL,
    case_title        TEXT        NOT NULL,
    deleted_by        UUID        REFERENCES users(id),
    deleted_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    people_count      INTEGER     NOT NULL,
    life_events_count INTEGER     NOT NULL,
    documents_count   INTEGER     NOT NULL,
    files_count       INTEGER     NOT NULL
);

CREATE INDEX ON case_deletion_records(case_id);
