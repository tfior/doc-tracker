-- Add deleted_at to all soft-deletable entity tables

ALTER TABLE cases       ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE people      ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE claim_lines ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE life_events ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE documents   ADD COLUMN deleted_at TIMESTAMPTZ;


-- Indexes for trash queries (WHERE deleted_at IS NOT NULL)

CREATE INDEX ON cases(deleted_at)       WHERE deleted_at IS NOT NULL;
CREATE INDEX ON people(deleted_at)      WHERE deleted_at IS NOT NULL;
CREATE INDEX ON claim_lines(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX ON life_events(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX ON documents(deleted_at)   WHERE deleted_at IS NOT NULL;


-- Update FK constraints to ON DELETE CASCADE so permanent deletion of a parent
-- propagates to its children. Nullable FKs that point "upward" use SET NULL instead.

-- people → cases
ALTER TABLE people
    DROP CONSTRAINT people_case_id_fkey,
    ADD CONSTRAINT people_case_id_fkey
        FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE;

-- cases → people (nullable; SET NULL when root person is hard-deleted)
ALTER TABLE cases
    DROP CONSTRAINT cases_primary_root_person_id_fkey,
    ADD CONSTRAINT cases_primary_root_person_id_fkey
        FOREIGN KEY (primary_root_person_id) REFERENCES people(id) ON DELETE SET NULL;

-- person_relationships → people and cases
ALTER TABLE person_relationships
    DROP CONSTRAINT person_relationships_person_id_fkey,
    ADD CONSTRAINT person_relationships_person_id_fkey
        FOREIGN KEY (person_id) REFERENCES people(id) ON DELETE CASCADE;

ALTER TABLE person_relationships
    DROP CONSTRAINT person_relationships_parent_id_fkey,
    ADD CONSTRAINT person_relationships_parent_id_fkey
        FOREIGN KEY (parent_id) REFERENCES people(id) ON DELETE CASCADE;

ALTER TABLE person_relationships
    DROP CONSTRAINT person_relationships_case_id_fkey,
    ADD CONSTRAINT person_relationships_case_id_fkey
        FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE;

-- claim_lines → cases and people
ALTER TABLE claim_lines
    DROP CONSTRAINT claim_lines_case_id_fkey,
    ADD CONSTRAINT claim_lines_case_id_fkey
        FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE;

ALTER TABLE claim_lines
    DROP CONSTRAINT claim_lines_root_person_id_fkey,
    ADD CONSTRAINT claim_lines_root_person_id_fkey
        FOREIGN KEY (root_person_id) REFERENCES people(id) ON DELETE CASCADE;

-- life_events → cases and people
ALTER TABLE life_events
    DROP CONSTRAINT life_events_case_id_fkey,
    ADD CONSTRAINT life_events_case_id_fkey
        FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE;

ALTER TABLE life_events
    DROP CONSTRAINT life_events_person_id_fkey,
    ADD CONSTRAINT life_events_person_id_fkey
        FOREIGN KEY (person_id) REFERENCES people(id) ON DELETE CASCADE;

-- documents → cases, people, life_events
ALTER TABLE documents
    DROP CONSTRAINT documents_case_id_fkey,
    ADD CONSTRAINT documents_case_id_fkey
        FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE;

ALTER TABLE documents
    DROP CONSTRAINT documents_person_id_fkey,
    ADD CONSTRAINT documents_person_id_fkey
        FOREIGN KEY (person_id) REFERENCES people(id) ON DELETE CASCADE;

-- life_event_id is nullable; SET NULL so a document survives life event deletion
ALTER TABLE documents
    DROP CONSTRAINT documents_life_event_id_fkey,
    ADD CONSTRAINT documents_life_event_id_fkey
        FOREIGN KEY (life_event_id) REFERENCES life_events(id) ON DELETE SET NULL;

-- file_attachments → documents
ALTER TABLE file_attachments
    DROP CONSTRAINT file_attachments_document_id_fkey,
    ADD CONSTRAINT file_attachments_document_id_fkey
        FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE;
