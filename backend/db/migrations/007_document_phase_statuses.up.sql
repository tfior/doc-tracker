-- Replace the single document status model with four per-phase status columns.
-- All existing document and file_attachment rows are cleared; reseed after running this migration.

-- Clear documents first (drops FK dependency on document_statuses via status_id).
TRUNCATE TABLE documents CASCADE;

-- Drop the old single status column while the table is empty.
ALTER TABLE documents DROP COLUMN status_id;

-- Clear old system statuses (pending / collected / verified / unobtainable).
DELETE FROM document_statuses;

-- Add phase column. Table is empty so NOT NULL requires no DEFAULT.
CREATE TYPE document_phase AS ENUM ('official_copy', 'amendment', 'apostille', 'translation', 'any');
ALTER TABLE document_statuses ADD COLUMN phase document_phase NOT NULL;

-- Convert system_key from the old enum type to plain text.
ALTER TABLE document_statuses ALTER COLUMN system_key TYPE text USING system_key::text;
DROP TYPE document_system_key;

-- Insert new system statuses.
INSERT INTO document_statuses (label, phase, is_system, system_key, progress_bucket) VALUES
    ('Not Started',            'official_copy', true, 'official_copy_default', 'not_started'),
    ('Not Started',            'apostille',     true, 'apostille_default',     'not_started'),
    ('Not Started',            'translation',   true, 'translation_default',   'not_started'),
    ('Unknown',                'amendment',     true, 'amendment_default',     'not_started'),
    ('Required — Not Started', 'amendment',     true, null,                    'not_started'),
    ('Researching',            'official_copy', true, null,                    'in_progress'),
    ('Researching',            'amendment',     true, null,                    'in_progress'),
    ('Requested',              'official_copy', true, null,                    'in_progress'),
    ('Requested',              'amendment',     true, null,                    'in_progress'),
    ('Sent',                   'apostille',     true, null,                    'in_progress'),
    ('Sent',                   'translation',   true, null,                    'in_progress'),
    ('Ready for Review',       'any',           true, null,                    'in_progress'),
    ('Complete',               'any',           true, null,                    'complete'),
    ('Not Required',           'any',           true, null,                    'complete');

-- Add four phase status columns to documents.
-- Table is empty so we can add as NOT NULL directly.
ALTER TABLE documents
    ADD COLUMN official_copy_status_id uuid NOT NULL REFERENCES document_statuses(id),
    ADD COLUMN amendment_status_id     uuid NOT NULL REFERENCES document_statuses(id),
    ADD COLUMN apostille_status_id     uuid NOT NULL REFERENCES document_statuses(id),
    ADD COLUMN translation_status_id   uuid NOT NULL REFERENCES document_statuses(id);
